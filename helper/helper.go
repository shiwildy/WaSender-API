package helper

import (
	"os"
	"context"
	"net/http"
	"fmt"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
	"go.mau.fi/whatsmeow/types"
)

type WaClientInfo struct {
	Client     *whatsmeow.Client
}

func Register(selfclient *whatsmeow.Client) *WaClientInfo {
	return &WaClientInfo{
		Client:  selfclient,
	}
}

func (BClient *WaClientInfo) SendMessage(sendto string, msgcontent string) error {
	sendtos := sendto + "@s.whatsapp.net"
	receivenum, _ := types.ParseJID(sendtos)
	_, err := BClient.Client.SendMessage(context.Background(), receivenum, &waProto.Message{
		Conversation: proto.String(msgcontent),
	})
	if err != nil {
		return err
	}

	return nil
}

func (BClient *WaClientInfo) SendImage(sendto, imagepath, caption string) error {
	sendtos := sendto + "@s.whatsapp.net"
	receivenum, _ := types.ParseJID(sendtos)

	// read files
	data, err := os.ReadFile(imagepath)
	if err != nil {
		return fmt.Errorf("failed to open files: %v", err)
	}

	// upload to whatsapp
	media, err := BClient.Client.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload media: %v", err)
	}

	// set to empty if caption is not field
	captions := ""
	if caption != "" {
		captions = caption
	}

	BClient.Client.SendMessage(context.Background(), receivenum, &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Mimetype:       proto.String(http.DetectContentType(data)),
			StaticURL:      proto.String(media.URL),
			DirectPath:     proto.String(media.DirectPath),
			MediaKey:       media.MediaKey,
			FileEncSHA256:  media.FileEncSHA256,
			FileSHA256:     media.FileSHA256,
			FileLength:     proto.Uint64(media.FileLength),
			Caption:        proto.String(captions),
		},
    })

	return nil
}

func (BClient *WaClientInfo) SendDocument(sendto, documentpath, caption string, filename string) error {
	sendtos := sendto + "@s.whatsapp.net"
	receivenum, _ := types.ParseJID(sendtos)

	// read files
	data, err := os.ReadFile(documentpath)
	if err != nil {
		return fmt.Errorf("failed to read document file: %v", err)
	}

	upmedia, err := BClient.Client.Upload(context.Background(), data, whatsmeow.MediaDocument)
	if err != nil {
		return fmt.Errorf("failed to upload document: %v", err)
	}
	// set to empty if caption is not field
	captions := ""
	if caption != "" {
		captions = caption
	}

	BClient.Client.SendMessage(context.Background(), receivenum, &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			URL:           proto.String(upmedia.URL),
			DirectPath:    proto.String(upmedia.DirectPath),
			FileLength:    proto.Uint64(upmedia.FileLength),
			FileName:      proto.String(filename),
			Caption:       proto.String(captions),
			MediaKey:      upmedia.MediaKey,
			FileEncSHA256: upmedia.FileEncSHA256,
			FileSHA256:    upmedia.FileSHA256,
		},
	})

	return nil
}
