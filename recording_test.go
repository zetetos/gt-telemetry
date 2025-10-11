package gttelemetry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordingFunctionality(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a client
	client, err := New(Options{
		Source:   "file://data/demo/demo.cast", // Use demo file if available
		LogLevel: "error",                      // Reduce noise during tests
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("StartRecording", func(t *testing.T) {
		// Test starting recording with .gtz file
		gtzPath := filepath.Join(tmpDir, "test_recording.gtz")
		err := client.StartRecording(gtzPath)
		if err != nil {
			t.Fatalf("Failed to start recording: %v", err)
		}

		// Check that recording is active
		if !client.IsRecording() {
			t.Error("IsRecording should return true after starting recording")
		}

		// Stop recording for cleanup
		err = client.StopRecording()
		if err != nil {
			t.Errorf("Failed to stop recording: %v", err)
		}
	})

	t.Run("StartRecordingPlainFile", func(t *testing.T) {
		// Test starting recording with .gtr file
		gtrPath := filepath.Join(tmpDir, "test_recording.gtr")
		err := client.StartRecording(gtrPath)
		if err != nil {
			t.Fatalf("Failed to start recording: %v", err)
		}

		// Check that recording is active
		if !client.IsRecording() {
			t.Error("IsRecording should return true after starting recording")
		}

		// Stop recording for cleanup
		err = client.StopRecording()
		if err != nil {
			t.Errorf("Failed to stop recording: %v", err)
		}
	})

	t.Run("InvalidFileExtension", func(t *testing.T) {
		// Test with invalid file extension
		invalidPath := filepath.Join(tmpDir, "test_recording.txt")
		err := client.StartRecording(invalidPath)
		if err == nil {
			t.Error("Expected error for invalid file extension, got nil")
			_ = client.StopRecording() // Cleanup just in case
		}
	})

	t.Run("AlreadyRecording", func(t *testing.T) {
		// Start recording
		gtzPath := filepath.Join(tmpDir, "test_recording2.gtz")
		err := client.StartRecording(gtzPath)
		if err != nil {
			t.Fatalf("Failed to start first recording: %v", err)
		}

		// Try to start another recording
		gtzPath2 := filepath.Join(tmpDir, "test_recording3.gtz")
		err = client.StartRecording(gtzPath2)
		if err == nil {
			t.Error("Expected error when trying to start recording while already recording")
		}

		// Stop the first recording
		err = client.StopRecording()
		if err != nil {
			t.Errorf("Failed to stop recording: %v", err)
		}
	})

	t.Run("StopWhenNotRecording", func(t *testing.T) {
		// Make sure we're not recording
		if client.IsRecording() {
			_ = client.StopRecording()
		}

		// Try to stop recording when not recording
		err := client.StopRecording()
		if err == nil {
			t.Error("Expected error when trying to stop recording while not recording")
		}
	})

	t.Run("IsRecording", func(t *testing.T) {
		// Should not be recording initially
		if client.IsRecording() {
			t.Error("IsRecording should return false initially")
		}

		// Start recording
		gtzPath := filepath.Join(tmpDir, "test_recording4.gtz")
		err := client.StartRecording(gtzPath)
		if err != nil {
			t.Fatalf("Failed to start recording: %v", err)
		}

		// Should be recording now
		if !client.IsRecording() {
			t.Error("IsRecording should return true after starting recording")
		}

		// Stop recording
		err = client.StopRecording()
		if err != nil {
			t.Errorf("Failed to stop recording: %v", err)
		}

		// Should not be recording after stopping
		if client.IsRecording() {
			t.Error("IsRecording should return false after stopping recording")
		}
	})

	t.Run("FileCreation", func(t *testing.T) {
		// Test that files are actually created
		gtzPath := filepath.Join(tmpDir, "test_file_creation.gtz")
		gtrPath := filepath.Join(tmpDir, "test_file_creation.gtr")

		// Test .gtz file creation
		err := client.StartRecording(gtzPath)
		if err != nil {
			t.Fatalf("Failed to start gtz recording: %v", err)
		}

		// Give it a moment to create the file
		time.Sleep(10 * time.Millisecond)

		err = client.StopRecording()
		if err != nil {
			t.Errorf("Failed to stop gtz recording: %v", err)
		}

		// Check if file exists
		if _, err := os.Stat(gtzPath); os.IsNotExist(err) {
			t.Error("GZT file was not created")
		}

		// Test .gtr file creation
		err = client.StartRecording(gtrPath)
		if err != nil {
			t.Fatalf("Failed to start gtr recording: %v", err)
		}

		// Give it a moment to create the file
		time.Sleep(10 * time.Millisecond)

		err = client.StopRecording()
		if err != nil {
			t.Errorf("Failed to stop gtr recording: %v", err)
		}

		// Check if file exists
		if _, err := os.Stat(gtrPath); os.IsNotExist(err) {
			t.Error("GTR file was not created")
		}
	})
}
