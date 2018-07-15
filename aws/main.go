package main

import (
	"fmt"
	"log"
	"os"

	logger "github.com/cdimascio/go-bunyan-logger"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/gocql/gocql"
)

var queueURL = "https://sqs.eu-west-1.amazonaws.com/355555488900/TomsTestQueue"

//Log holds ref to bunyan logger
var Log = InitializeLogger()

//AmazonWebServices provides a single point of contact with AWS SDK
type AmazonWebServices struct {
	Config   *aws.Config
	Session  *session.Session
	SQS      sqsiface.SQSAPI
	queueURL string
}

//InitializeLogger creates a new log instance
func InitializeLogger() *logger.Logger {
	var logInstance = logger.NewLogger("AWSExampleAPP")
	var envLogLevel, setFlag = os.LookupEnv("logLevel")

	levelMap := map[string]logger.Level{
		"fatal": logger.LevelFatal,
		"error": logger.LevelError,
		"warn":  logger.LevelWarn,
		"info":  logger.LevelInfo,
		"debug": logger.LevelDebug,
		"trace": logger.LevelTrace,
	}

	if setFlag && levelMap[envLogLevel] != 0 {
		logInstance.SetLevel(levelMap[envLogLevel])
	} else {
		logInstance.SetLevel(logger.LevelTrace)
	}
	return logInstance
}

//InitializeAWS create AWS Session
func InitializeAWS(queueURL string) AmazonWebServices {
	config := &aws.Config{
		Region: aws.String("eu-west-1"),
	}
	session := session.Must(session.NewSession(config))
	SQS := sqs.New(session)
	return AmazonWebServices{config, session, SQS, queueURL}
}

//SendMessage sends a message to SQS
func (AWS *AmazonWebServices) SendMessage() {
	message := sqs.SendMessageInput{
		MessageBody: aws.String("Hello, World!"),
		QueueUrl:    &AWS.queueURL,
	}
	_, err := AWS.SQS.SendMessage(&message)
	if err != nil {
		Log.Errorf("Unable To Send Message To SQS")
		return
	}
	Log.Info("Sent message to SQS")
}

//ReadMessageIntoChannel reads a message from SQS
func (AWS *AmazonWebServices) ReadMessageIntoChannel(channel chan *sqs.Message) {
	options := sqs.ReceiveMessageInput{
		QueueUrl:            &AWS.queueURL,
		MaxNumberOfMessages: aws.Int64(1),
	}
	result, err := AWS.SQS.ReceiveMessage(&options)
	if err != nil {
		Log.Errorf("Unable To Read Message From SQS")
		return
	}
	if len(result.Messages) == 0 {
		Log.Info("Zero Messages Received")
		return
	}
	Log.Info("Got message from SQS")
	channel <- result.Messages[0]
}

//Cassandra provides a single point of contact with gocql client
type Cassandra struct {
	Session *gocql.Session
}

//CassandraClient provides a decoupled interface for unit testing
type CassandraClient interface {
	SaveItem(string, string) error
}

//SaveItem defines interface method for saving items.
func (Cass Cassandra) SaveItem(messageID string, body string) error {
	return Cass.Session.Query("INSERT INTO tom.test (id, body) VALUES (?, ?)", messageID, body).Exec()
}

//IntializeCassandra creates session for reuse
func IntializeCassandra() Cassandra {
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "tom"
	cluster.Consistency = gocql.Quorum
	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}
	return Cassandra{session}
}

//PersistMessage takes a message via a channel and persists it to Cassandra
func PersistMessage(Cass CassandraClient, channel chan *sqs.Message) {
	message := <-channel
	messageID := *message.MessageId
	body := *message.Body

	err := Cass.SaveItem(messageID, body)

	if err != nil {
		Log.Errorf("Unable To Save To Cassandra")
	}
	Log.Info("Message Saved To Cassandra")
}

func main() {
	AWS := InitializeAWS(queueURL)

	Cassandra := IntializeCassandra()
	defer Cassandra.Session.Close()

	messageChannel := make(chan *sqs.Message)
	defer close(messageChannel)

	go func() {
		for {
			AWS.SendMessage()
		}
	}()

	go func() {
		for {
			AWS.ReadMessageIntoChannel(messageChannel)
		}
	}()

	go func() {
		for {
			PersistMessage(Cassandra, messageChannel)
		}
	}()

	var input string
	fmt.Scanln(&input)
}
