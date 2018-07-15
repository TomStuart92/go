package main

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	. "github.com/franela/goblin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type FakeSQS struct {
	mock.Mock
	sqsiface.SQSAPI
}

func (SQS *FakeSQS) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	args := SQS.Called(input)
	return args.Get(0).(*sqs.SendMessageOutput), args.Error(1)
}

func (SQS *FakeSQS) ReceiveMessage(input *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	args := SQS.Called(input)
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

type FakeCassandra struct {
	mock.Mock
}

func (Session *FakeCassandra) SaveItem(messageID string, body string) error {
	args := Session.Called(messageID, body)
	return args.Error(0)
}

func Test(t *testing.T) {
	g := Goblin(t)

	g.Describe("Initializing Cassandra", func() {
		g.It("Should return a Cassandra Struct with Session ", func() {
			testResult := IntializeCassandra()
			g.Assert(testResult.Session != nil).IsTrue()
		})
	})

	g.Describe("Initializing AWS", func() {
		g.It("Should return an AWS Struct", func() {
			QueueURL := "Fake URL"
			testResult := InitializeAWS(QueueURL)
			g.Assert(testResult.queueURL).Equal(QueueURL)
			g.Assert(testResult.Session != nil).IsTrue()
			g.Assert(testResult.Config != nil).IsTrue()
			g.Assert(testResult.SQS != nil).IsTrue()
		})
	})

	g.Describe("Send Messages", func() {
		g.It("Should call AWS.SQS.SendMessage", func() {
			testAWS := new(AmazonWebServices)
			fakeSQS := FakeSQS{}
			response := new(sqs.SendMessageOutput)
			fakeSQS.On("SendMessage", mock.Anything).Return(response, nil)
			testAWS.SQS = &fakeSQS
			testAWS.SendMessage()
			fakeSQS.AssertExpectations(t)
		})
		g.It("Should Not Throw If An Error Occurs", func() {
			testAWS := new(AmazonWebServices)
			fakeSQS := FakeSQS{}
			sampleErr := errors.New("Something Went Wrong With AWS.SQS Send")
			response := new(sqs.SendMessageOutput)
			fakeSQS.On("SendMessage", mock.Anything).Return(response, sampleErr)
			testAWS.SQS = &fakeSQS
			testAWS.SendMessage()
			fakeSQS.AssertExpectations(t)
		})
	})

	g.Describe("Receive Messages", func() {
		g.It("Should call AWS.SQS.SendMessage", func() {
			testAWS := new(AmazonWebServices)
			fakeSQS := FakeSQS{}

			response := sqs.ReceiveMessageOutput{
				Messages: []*sqs.Message{
					{Body: aws.String(`{"from":"user_1","to":"room_1","msg":"Hello!"}`)},
				},
			}
			fakeSQS.On("ReceiveMessage", mock.Anything).Return(&response, nil)

			testAWS.SQS = &fakeSQS
			messageChannel := make(chan *sqs.Message, 1)

			testAWS.ReadMessageIntoChannel(messageChannel)
			fakeSQS.AssertExpectations(t)
			close(messageChannel)
		})

		g.It("Should Place Messages Into Queue", func() {
			testAWS := new(AmazonWebServices)
			fakeSQS := FakeSQS{}
			response := sqs.ReceiveMessageOutput{
				Messages: []*sqs.Message{
					{Body: aws.String(`{"from":"user_1","to":"room_1","msg":"Hello!"}`)},
				},
			}
			fakeSQS.On("ReceiveMessage", mock.Anything).Return(&response, nil)
			testAWS.SQS = &fakeSQS
			messageChannel := make(chan *sqs.Message, 1)
			testAWS.ReadMessageIntoChannel(messageChannel)
			message := <-messageChannel
			assert.Equal(t, response.Messages[0].Body, message.Body)
			close(messageChannel)
		})

		g.It("Should Not Throw If An Error Occurs", func() {
			testAWS := new(AmazonWebServices)
			fakeSQS := FakeSQS{}

			sampleErr := errors.New("Something Went Wrong With AWS.SQS Read")
			response := new(sqs.ReceiveMessageOutput)

			fakeSQS.On("ReceiveMessage", mock.Anything).Return(response, sampleErr)

			testAWS.SQS = &fakeSQS
			messageChannel := make(chan *sqs.Message, 1)
			testAWS.ReadMessageIntoChannel(messageChannel)
			fakeSQS.AssertExpectations(t)
			close(messageChannel)
		})

		g.It("Should Not Throw On No Messages", func() {
			testAWS := new(AmazonWebServices)
			fakeSQS := FakeSQS{}

			response := new(sqs.ReceiveMessageOutput)
			fakeSQS.On("ReceiveMessage", mock.Anything).Return(response, nil)

			testAWS.SQS = &fakeSQS
			messageChannel := make(chan *sqs.Message, 1)

			testAWS.ReadMessageIntoChannel(messageChannel)
			fakeSQS.AssertExpectations(t)
			close(messageChannel)
		})
	})

	g.Describe("Persist Message", func() {
		g.It("Should Call Cassandra.SaveItem", func() {
			FakeCassandra := new(FakeCassandra)
			messages := []*sqs.Message{
				{
					Body:      aws.String(`Hello`),
					MessageId: aws.String(`123`),
				},
			}

			FakeCassandra.On("SaveItem", *messages[0].MessageId, *messages[0].Body).Return(nil)
			messageChannel := make(chan *sqs.Message, 1)
			messageChannel <- messages[0]

			PersistMessage(FakeCassandra, messageChannel)
			FakeCassandra.AssertExpectations(t)
			close(messageChannel)
		})

		g.It("Should Not Throw on Error", func() {
			FakeCassandra := new(FakeCassandra)

			messages := []*sqs.Message{
				{
					Body:      aws.String(`Hello`),
					MessageId: aws.String(`123`),
				},
			}

			sampleErr := errors.New("Something Went Wrong With Cassandra")
			FakeCassandra.On("SaveItem", *messages[0].MessageId, *messages[0].Body).Return(sampleErr)

			messageChannel := make(chan *sqs.Message, 1)
			messageChannel <- messages[0]

			PersistMessage(FakeCassandra, messageChannel)
			FakeCassandra.AssertExpectations(t)
			close(messageChannel)
		})
	})
}
