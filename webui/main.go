package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	_ "github.com/lib/pq"
)

func connectToDatabase() (*sql.DB, error) {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres dbname=dbec2 password=password sslmode=disable")
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	fmt.Println("Database is running!")
	return db, nil
}

func createEC2Instance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-north-1"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	svc := ec2.New(sess)

	runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
		ImageId:      aws.String("ami-0cf72be2f86b04e9b"),
		InstanceType: aws.String("t3.micro"),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	instanceID := *runResult.Instances[0].InstanceId

	db, err := connectToDatabase()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO instances (instance_id) VALUES ($1)", instanceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer db.Close()

	json.NewEncoder(w).Encode(struct{ InstanceID string }{InstanceID: instanceID})

}

func terminateEC2Instance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-north-1")},
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	svc := ec2.New(sess)

	decoder := json.NewDecoder(r.Body)
	var instanceID struct {
		InstanceID string `json:"instanceId"`
	}
	err = decoder.Decode(&instanceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(instanceID.InstanceID)},
	}

	result, err := svc.TerminateInstances(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db, err := connectToDatabase()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("DELETE FROM instances WHERE instance_id = $1", instanceID.InstanceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"terminated": len(result.TerminatingInstances) > 0,
	}
	defer db.Close()
	json.NewEncoder(w).Encode(response)

}

func listEC2Instances(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("eu-north-1"),
	})

	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	svc := ec2.New(sess)

	result, err := svc.DescribeInstances(nil)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	instances := []string{}
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, *instance.InstanceId)
		}
	}

	json.NewEncoder(w).Encode(instances)
}

func main() {

	http.HandleFunc("/api/ec2/create", createEC2Instance)
	http.HandleFunc("/api/ec2/terminate", terminateEC2Instance)
	http.HandleFunc("/api/ec2/list", listEC2Instances)
	fs := http.FileServer(http.Dir("html"))
	http.Handle("/", fs)

	// Start the server on port 8080
	http.ListenAndServe(":8080", nil)

}
