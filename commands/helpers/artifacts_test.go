package helpers

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

const (
	artifactsTestArchivedFile  = "archive_file"
	artifactsTestArchivedFile2 = "archive_file2"
)

type testNetwork struct {
	common.MockNetwork
	downloadState  common.DownloadState
	downloadCalled int
	uploadState    common.UploadState
	uploadCalled   int
	uploadFormat   common.ArtifactFormat
	uploadName     string
	uploadType     string
	uploadedFiles  []string
}

func (m *testNetwork) DownloadArtifacts(config common.JobCredentials, artifactsFile string) common.DownloadState {
	m.downloadCalled++

	if m.downloadState == common.DownloadSucceeded {
		file, err := os.Create(artifactsFile)
		if err != nil {
			logrus.Warningln(err)
			return common.DownloadFailed
		}
		defer file.Close()

		archive := zip.NewWriter(file)
		archive.Create(artifactsTestArchivedFile)
		archive.Close()
	}
	return m.downloadState
}

func (m *testNetwork) consumeZipUpload(config common.JobCredentials, reader io.Reader, options common.ArtifactsOptions) common.UploadState {
	var buffer bytes.Buffer
	io.Copy(&buffer, reader)
	archive, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		logrus.Warningln(err)
		return common.UploadForbidden
	}

	for _, file := range archive.File {
		m.uploadedFiles = append(m.uploadedFiles, file.Name)
	}

	m.uploadFormat = common.ArtifactFormatZip

	return m.uploadState
}

func (m *testNetwork) consumeGzipUpload(config common.JobCredentials, reader io.Reader, options common.ArtifactsOptions) common.UploadState {
	var buffer bytes.Buffer
	io.Copy(&buffer, reader)

	stream := bytes.NewReader(buffer.Bytes())

	gz, err := gzip.NewReader(stream)
	gz.Multistream(false)
	if err != nil {
		logrus.Warningln("Invalid gzip stream")
		return common.UploadForbidden
	}

	// Read multiple streams
	for {
		_, err = io.Copy(ioutil.Discard, gz)
		if err != nil {
			logrus.Warningln("Invalid gzip stream")
			return common.UploadForbidden
		}

		m.uploadedFiles = append(m.uploadedFiles, gz.Name)

		if gz.Reset(stream) == io.EOF {
			break
		}
		gz.Multistream(false)
	}

	m.uploadFormat = common.ArtifactFormatGzip

	return m.uploadState
}

func (m *testNetwork) consumeRawUpload(config common.JobCredentials, reader io.Reader, options common.ArtifactsOptions) common.UploadState {
	io.Copy(ioutil.Discard, reader)

	m.uploadedFiles = append(m.uploadedFiles, "raw")
	m.uploadFormat = common.ArtifactFormatRaw
	return m.uploadState
}

func (m *testNetwork) UploadRawArtifacts(config common.JobCredentials, reader io.Reader, options common.ArtifactsOptions) common.UploadState {
	m.uploadCalled++

	if m.uploadState == common.UploadSucceeded {
		m.uploadType = options.Type
		m.uploadName = options.BaseName

		switch options.Format {
		case common.ArtifactFormatZip, common.ArtifactFormatDefault:
			return m.consumeZipUpload(config, reader, options)

		case common.ArtifactFormatGzip:
			return m.consumeGzipUpload(config, reader, options)

		case common.ArtifactFormatRaw:
			return m.consumeRawUpload(config, reader, options)

		default:
			return common.UploadForbidden
		}
	}

	return m.uploadState
}

func writeTestFile(t *testing.T, fileName string) {
	err := ioutil.WriteFile(fileName, nil, 0600)
	require.NoError(t, err, "Writing file:", fileName)
}
