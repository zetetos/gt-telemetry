package gttelemetry_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	gttelemetry "github.com/zetetos/gt-telemetry/v2"
)

type RecordingTestSuite struct {
	suite.Suite

	client *gttelemetry.Client
	tmpDir string
}

func TestRecordingTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RecordingTestSuite))
}

func (suite *RecordingTestSuite) SetupTest() {
	// Create a temporary directory for test files
	suite.tmpDir = suite.T().TempDir()

	// Create a client
	client, err := gttelemetry.New(gttelemetry.Options{
		Source:   "file://data/demo/demo.cast", // Use demo file if available
		LogLevel: "error",                      // Reduce noise during tests
	})
	suite.Require().NoError(err, "Failed to create client")
	suite.client = client
}

func (suite *RecordingTestSuite) TearDownTest() {
	// Make sure we stop recording if still active
	if suite.client != nil && suite.client.IsRecording() {
		_ = suite.client.StopRecording()
	}
}

func (suite *RecordingTestSuite) TestStartRecording() {
	// Test starting recording with .gtz file
	gtzPath := filepath.Join(suite.tmpDir, "test_recording.gtz")

	err := suite.client.StartRecording(gtzPath)
	suite.Require().NoError(err, "Failed to start recording")

	// Check that recording is active
	suite.True(suite.client.IsRecording(), "IsRecording should return true after starting recording")

	// Stop recording for cleanup
	err = suite.client.StopRecording()
	suite.NoError(err, "Failed to stop recording")
}

func (suite *RecordingTestSuite) TestStartRecordingPlainFile() {
	// Test starting recording with .gtr file
	gtrPath := filepath.Join(suite.tmpDir, "test_recording.gtr")

	err := suite.client.StartRecording(gtrPath)
	suite.Require().NoError(err, "Failed to start recording")

	// Check that recording is active
	suite.True(suite.client.IsRecording(), "IsRecording should return true after starting recording")

	// Stop recording for cleanup
	err = suite.client.StopRecording()
	suite.NoError(err, "Failed to stop recording")
}

func (suite *RecordingTestSuite) TestInvalidFileExtension() {
	// Test with invalid file extension
	invalidPath := filepath.Join(suite.tmpDir, "test_recording.txt")

	err := suite.client.StartRecording(invalidPath)
	suite.Require().Error(err, "Expected error for invalid file extension, got nil")

	// Cleanup just in case
	if suite.client.IsRecording() {
		_ = suite.client.StopRecording()
	}
}

func (suite *RecordingTestSuite) TestAlreadyRecording() {
	// Start recording
	gtzPath := filepath.Join(suite.tmpDir, "test_recording2.gtz")

	err := suite.client.StartRecording(gtzPath)
	suite.Require().NoError(err, "Failed to start first recording")

	// Try to start another recording
	gtzPath2 := filepath.Join(suite.tmpDir, "test_recording3.gtz")

	err = suite.client.StartRecording(gtzPath2)
	suite.Require().Error(err, "Expected error when trying to start recording while already recording")

	// Stop the first recording
	err = suite.client.StopRecording()
	suite.NoError(err, "Failed to stop recording")
}

func (suite *RecordingTestSuite) TestStopWhenNotRecording() {
	// Make sure we're not recording
	if suite.client.IsRecording() {
		_ = suite.client.StopRecording()
	}

	// Try to stop recording when not recording
	err := suite.client.StopRecording()
	suite.Error(err, "Expected error when trying to stop recording while not recording")
}

func (suite *RecordingTestSuite) TestIsRecording() {
	// Should not be recording initially
	suite.False(suite.client.IsRecording(), "IsRecording should return false initially")

	// Start recording
	gtzPath := filepath.Join(suite.tmpDir, "test_recording4.gtz")

	err := suite.client.StartRecording(gtzPath)
	suite.Require().NoError(err, "Failed to start recording")

	// Should be recording now
	suite.True(suite.client.IsRecording(), "IsRecording should return true after starting recording")

	// Stop recording
	err = suite.client.StopRecording()
	suite.Require().NoError(err, "Failed to stop recording")

	// Should not be recording after stopping
	suite.False(suite.client.IsRecording(), "IsRecording should return false after stopping recording")
}

func (suite *RecordingTestSuite) TestFileCreation() {
	// Test that files are actually created
	gtzPath := filepath.Join(suite.tmpDir, "test_file_creation.gtz")
	gtrPath := filepath.Join(suite.tmpDir, "test_file_creation.gtr")

	// Test .gtz file creation
	err := suite.client.StartRecording(gtzPath)
	suite.Require().NoError(err, "Failed to start gtz recording")

	// Give it a moment to create the file
	time.Sleep(10 * time.Millisecond)

	err = suite.client.StopRecording()
	suite.Require().NoError(err, "Failed to stop gtz recording")

	// Check if file exists
	_, err = os.Stat(gtzPath)
	suite.False(os.IsNotExist(err), "GZT file was not created")

	// Test .gtr file creation
	err = suite.client.StartRecording(gtrPath)
	suite.Require().NoError(err, "Failed to start gtr recording")

	// Give it a moment to create the file
	time.Sleep(10 * time.Millisecond)

	err = suite.client.StopRecording()
	suite.Require().NoError(err, "Failed to stop gtr recording")

	// Check if file exists
	_, err = os.Stat(gtrPath)
	suite.False(os.IsNotExist(err), "GTR file was not created")
}
