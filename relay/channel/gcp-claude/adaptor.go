package gcp_claude

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"one-api/dto"
	"one-api/relay/channel"
	relaycommon "one-api/relay/common"
	"strings"
)

const (
	// LOCATION europe-west1 or us-east5
	LOCATION = "us-east5"
)

type Adaptor struct {
}

func (a *Adaptor) Init(info *relaycommon.RelayInfo) {
}

func (a *Adaptor) GetRequestURL(info *relaycommon.RelayInfo) (string, error) {
	parts := strings.SplitN(info.ApiKey, "|", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid api key: %s", info.ApiKey)
	}
	projectId := strings.TrimSpace(parts[0])
	model, err := getRedirectModel(info.UpstreamModelName)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:streamRawPredict", LOCATION, projectId, LOCATION, model), nil
}

func (a *Adaptor) SetupRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	channel.SetupApiRequestHeader(info, c, req)
	parts := strings.SplitN(info.ApiKey, "|", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid api key: %s", info.ApiKey)
	}
	json := strings.TrimSpace(parts[1])
	accessToken, err := getAccessToken(info.Proxy, json)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return nil
}

func (a *Adaptor) ConvertRequest(c *gin.Context, info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest) (any, error) {
	if request == nil {
		return nil, errors.New("request is nil")
	}
	return requestOpenAI2GcpClaude(*request)
}

func (a *Adaptor) ConvertAudioRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.AudioRequest) (io.Reader, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *Adaptor) ConvertRerankRequest(c *gin.Context, relayMode int, request dto.RerankRequest) (any, error) {
	return nil, nil
}

func (a *Adaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoApiRequest(a, c, info, requestBody)
}

func (a *Adaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (usage *dto.Usage, err *dto.OpenAIErrorWithStatusCode) {
	if info.IsStream {
		err, usage = gcpClaudeStreamHandler(c, resp, info)
	} else {
		err, usage = gcpClaudeHandler(c, resp)
	}
	return
}

func (a *Adaptor) GetModelList() (models []string) {
	for n := range modelIdMap {
		models = append(models, n)
	}
	return
}

func (a *Adaptor) GetChannelName() string {
	return ChannelName
}
