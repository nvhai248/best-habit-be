package sendnotificationprovider

import (
	"context"
	"encoding/json"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"google.golang.org/api/option"
)

type NotificationProvider interface {
	SendNotification(deviceToken string, title, body string) error
}

// NotificationService represents a service to handle push notifications
type NotificationService struct {
	app    *firebase.App
	ctx    context.Context
	client *messaging.Client
}

// NewNotificationService creates a new instance of NotificationService
func NewNotificationService(ctx context.Context, projectId, privateKeyId, privateKey, clientEmail, clientId string) (*NotificationService, error) {
	config := &firebase.Config{
		ProjectID: projectId,
		AuthOverride: &map[string]interface{}{
			"uid": clientEmail,
		},
	}

	type DataConfig struct {
		Type                    string `json:"type"`
		ProjectId               string `json:"project_id"`
		PrivateKeyId            string `json:"private_key_id"`
		PrivateKey              string `json:"private_key"`
		ClientEmail             string `json:"client_email"`
		ClientId                string `json:"client_id"`
		AuthUri                 string `json:"auth_uri"`
		TokenUri                string `json:"token_uri"`
		AuthProviderX509CertUrl string `json:"auth_provider_x509_cert_url"`
		ClientX509CertUrl       string `json:"client_x509_cert_url"`
		UniverseDomain          string `json:"universe_domain"`
	}

	dataConfig := DataConfig{
		Type:                    "service_account",
		ProjectId:               projectId,
		PrivateKeyId:            privateKeyId,
		PrivateKey:              privateKey,
		ClientEmail:             clientEmail,
		ClientId:                clientId,
		AuthUri:                 "https://accounts.google.com/o/oauth2/auth",
		TokenUri:                "https://oauth2.googleapis.com/token",
		AuthProviderX509CertUrl: "https://www.googleapis.com/oauth2/v1/certs",
		ClientX509CertUrl:       "https://www.googleapis.com/robot/v1/metadata/x509/firebase-adminsdk-c3by4%40best-habit.iam.gserviceaccount.com",
		UniverseDomain:          "googleapis.com",
	}

	jsonString, err := json.Marshal(dataConfig)
	if err != nil {
		return nil, err
	}

	opt := option.WithCredentialsJSON([]byte(jsonString))
	app, err := firebase.NewApp(ctx, config, opt)

	if err != nil {
		return nil, err
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}

	return &NotificationService{
		app:    app,
		ctx:    ctx,
		client: client,
	}, nil
}

// SendNotification sends a push notification to the specified device token
func (ns *NotificationService) SendNotification(deviceToken string, title, body string) error {
	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Token: deviceToken,
	}

	_, err := ns.client.Send(ns.ctx, message)
	if err != nil {
		return err
	}

	return nil
}

type NoOpNotificationProvider struct{}

func NewNoOpNotificationService() *NoOpNotificationProvider {
	return &NoOpNotificationProvider{}
}

func (ns *NoOpNotificationProvider) SendNotification(deviceToken string, title, body string) error {
	return nil
}
