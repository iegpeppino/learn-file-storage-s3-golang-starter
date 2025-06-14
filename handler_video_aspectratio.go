package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {

	type VideoData struct {
		Streams []struct {
			Index              int     `json:"index"`
			CodecName          string  `json:"codec_name"`
			CodecLongName      string  `json:"codec_long_name"`
			CodecType          string  `json:"codec_type"`
			Width              float64 `json:"width"`
			Height             float64 `json:"height"`
			DisplayAspectRatio string  `json:"display_aspect_ratio"`
		} `json:"streams"`
	}

	cmdPointer := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	// Initialize buffer and send stdout of command pointer to the buffer
	var vidJSON bytes.Buffer
	cmdPointer.Stdout = &vidJSON

	// Run command
	err := cmdPointer.Run()
	if err != nil {
		return "", fmt.Errorf("couldn't run video data command %w", err)
	}

	// Store unmarshaled data into struct
	var videoMetaData VideoData
	json.Unmarshal(vidJSON.Bytes(), &videoMetaData)

	aspectRatio := videoMetaData.Streams[0].Width / videoMetaData.Streams[0].Height

	landscapeUpper := 1.866666667  // 16/9 + 5%
	landscapeBottom := 1.688888889 // 16/9 - 5%

	portraitUpper := 0.590625  // 6/19 + 5%
	portraitBottom := 0.534375 // 6/19 - 5%

	if aspectRatio >= landscapeBottom && aspectRatio <= landscapeUpper {
		return "19:6", nil
	}

	if aspectRatio >= portraitBottom && aspectRatio <= portraitUpper {
		return "6:19", nil
	}

	return "other", nil

}

func processVideoForFastStart(filePath string) (string, error) {

	newPath := filePath + ".processing"

	cmdPointer := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", newPath)

	err := cmdPointer.Run()
	if err != nil {
		return "", fmt.Errorf("couldn't run ffmpeg command %w", err)
	}

	return newPath, nil
}
