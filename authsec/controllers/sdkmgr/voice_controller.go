package sdkmgr

import (
	"net/http"

	sdkmgrSvc "github.com/authsec-ai/authsec/services/sdkmgr"
	"github.com/gin-gonic/gin"
)

// VoiceController handles voice/chat authentication endpoints.
type VoiceController struct {
	svc *sdkmgrSvc.VoiceClientService
}

// NewVoiceController creates a new voice controller.
func NewVoiceController(svc *sdkmgrSvc.VoiceClientService) *VoiceController {
	return &VoiceController{svc: svc}
}

type interactRequest struct {
	Text      string `json:"text" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	AuthReqID string `json:"auth_req_id"`
	ClientID  string `json:"client_id"`
}

type pollRequest struct {
	AuthReqID string `json:"auth_req_id" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	ClientID  string `json:"client_id"`
}

type ttsRequest struct {
	Text  string `json:"text" binding:"required"`
	Voice string `json:"voice"`
}

// Interact handles POST /voice/interact.
func (vc *VoiceController) Interact(c *gin.Context) {
	var req interactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := vc.svc.Interact(req.Text, req.Email, req.AuthReqID, req.ClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Poll handles POST /voice/poll.
func (vc *VoiceController) Poll(c *gin.Context) {
	var req pollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := vc.svc.PollStatus(req.AuthReqID, req.Email, req.ClientID)
	c.JSON(http.StatusOK, result)
}

// TTS handles POST /voice/tts.
func (vc *VoiceController) TTS(c *gin.Context) {
	var req ttsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	audio, err := vc.svc.GenerateSpeech(req.Text, req.Voice)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "audio/mpeg", audio)
}
