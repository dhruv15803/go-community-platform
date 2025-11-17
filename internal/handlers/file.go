package handlers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (h *Handler) UserImageFileUploadHandler(w http.ResponseWriter, r *http.Request) {

	userId, ok := r.Context().Value(AuthUserId).(int)
	if !ok {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, err := h.storage.Users.GetUserById(userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, "user not found", http.StatusNotFound)
			return
		} else {
			writeJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}

	file, fileHeader, err := r.FormFile("imageFile")
	if err != nil {
		writeJSONError(w, "imageFile not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	log.Println("uploading file ", fileHeader.Filename)
	log.Println("file size ", fileHeader.Size)

	// store file temporarily on go server for retries during failure
	imageFileUploadsDir := "./uploads"
	uniqueNumSequence := int(time.Now().Unix())
	uniqueNumSequenceStr := strconv.Itoa(uniqueNumSequence)
	uniqueImageFileName := uniqueNumSequenceStr + "_" + fileHeader.Filename
	imageFileUploadPath := imageFileUploadsDir + "/" + uniqueImageFileName

	dest, err := os.Create(imageFileUploadPath)
	if err != nil {
		log.Println("failed creating local upload on server")
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer dest.Close()

	_, err = io.Copy(dest, file)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	file.Close()
	dest.Close()
	// all open files are closed here
	// image is copied into the destination image file path

	uploadFile, err := os.Open(imageFileUploadPath)
	if err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer uploadFile.Close()

	// s3 bucket image file path
	// /uploads/userId-123/uniqueImageFileName
	awsS3ObjectKey := fmt.Sprintf("%s/userId-%d/%s", "uploads", user.Id, uniqueImageFileName)

	maxRetries := 3
	isUploaded := false

	for i := 0; i < maxRetries; i++ {

		_, err = uploadFile.Seek(0, 0)
		if err != nil {
			break
		}

		_, err = h.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(os.Getenv("AWS_S3_BUCKET")),
			Key:    aws.String(awsS3ObjectKey),
			Body:   uploadFile,
		})

		if err != nil {
			log.Printf("failed uploading file %s to s3, attempt: %d\n", imageFileUploadPath, i+1)
			continue
		}

		isUploaded = true
		break
	}

	if !isUploaded {
		log.Printf("failed uploading file: %v\n", err)
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	uploadFile.Close()

	// remove file locally from server
	if err = os.Remove(imageFileUploadPath); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	uploadedObjectUrl := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", os.Getenv("AWS_S3_BUCKET"), os.Getenv("AWS_REGION"), awsS3ObjectKey)

	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Url     string `json:"url"`
	}

	if err := writeJSON(w, Response{Success: true, Message: "uploaded file successfully", Url: uploadedObjectUrl}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}
