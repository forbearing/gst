package aichat

import (
	modelaichat "github.com/forbearing/gst/internal/model/aichat"
	serviceaichat "github.com/forbearing/gst/internal/service/aichat"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/module"
	"github.com/forbearing/gst/service"
	"github.com/forbearing/gst/types/consts"
)

type (
	Model           = modelaichat.Model
	Provider        = modelaichat.Provider
	Chat            = modelaichat.Chat
	Message         = modelaichat.Message
	MessageFeedBack = modelaichat.MessageFeedback
)

// Register registers AI chat modules for managing AI providers, models, chats, and messages.
//
// Models:
//   - Model: AI model configuration and metadata
//   - Provider: AI provider configuration (OpenAI, Anthropic, Ollama, etc.)
//   - Chat: Chat conversation sessions
//   - Message: Messages within a chat conversation
//   - MessageFeedback: User feedback for messages
//
// Routes:
//
// Model module (full CRUD):
//   - POST     /api/models
//   - DELETE   /api/models/:id
//   - PUT      /api/models/:id
//   - PATCH    /api/models/:id
//   - GET      /api/models
//   - GET      /api/models/:id
//   - POST     /api/models/batch
//   - DELETE   /api/models/batch
//   - PUT      /api/models/batch
//   - PATCH    /api/models/batch
//
// Provider module (full CRUD):
//   - POST     /api/providers
//   - DELETE   /api/providers/:id
//   - PUT      /api/providers/:id
//   - PATCH    /api/providers/:id
//   - GET      /api/providers
//   - GET      /api/providers/:id
//   - POST     /api/providers/batch
//   - DELETE   /api/providers/batch
//   - PUT      /api/providers/batch
//   - PATCH    /api/providers/batch
//
// Chat module (full CRUD):
//   - POST     /api/ai/chats
//   - DELETE   /api/ai/chats/:chat_id
//   - PUT      /api/ai/chats/:chat_id
//   - PATCH    /api/ai/chats/:chat_id
//   - GET      /api/ai/chats
//   - GET      /api/ai/chats/:chat_id
//   - POST     /api/ai/chats/batch
//   - DELETE   /api/ai/chats/batch
//   - PUT      /api/ai/chats/batch
//   - PATCH    /api/ai/chats/batch
//
// Message module (full CRUD):
//   - POST     /api/ai/chats/:chat_id/messages
//   - DELETE   /api/ai/chats/:chat_id/messages/:msg_id
//   - PUT      /api/ai/chats/:chat_id/messages/:msg_id
//   - PATCH    /api/ai/chats/:chat_id/messages/:msg_id
//   - GET      /api/ai/chats/:chat_id/messages
//   - GET      /api/ai/chats/:chat_id/messages/:msg_id
//   - POST     /api/ai/chats/:chat_id/messages/batch
//   - DELETE   /api/ai/chats/:chat_id/messages/batch
//   - PUT      /api/ai/chats/:chat_id/messages/batch
//   - PATCH    /api/ai/chats/:chat_id/messages/batch
//
// MessageFeedback module (full CRUD):
//   - POST     /api/ai/chats/:chat_id/messages/:msg_id/feedback
//   - DELETE   /api/ai/chats/:chat_id/messages/:msg_id/feedback/:id
//   - PUT      /api/ai/chats/:chat_id/messages/:msg_id/feedback/:id
//   - PATCH    /api/ai/chats/:chat_id/messages/:msg_id/feedback/:id
//   - GET      /api/ai/chats/:chat_id/messages/:msg_id/feedback
//   - GET      /api/ai/chats/:chat_id/messages/:msg_id/feedback/:id
//
// TestConnection module:
//   - POST     /api/ai/providers/test-connection
//     Request body: Provider (with config information)
//     Response: TestConnectionRsp with success status and message
//
// ListModels module:
//   - POST     /api/ai/providers/models
//     Request body: Provider (with config information)
//     Response: ListModelsRsp with array of available models
//
// Supported provider types:
//   - openai: OpenAI API
//   - anthropic: Anthropic Claude API
//   - google: Google AI API
//   - azure: Azure OpenAI
//   - aliyun: Alibaba Cloud Tongyi
//   - baidu: Baidu Wenxin
//   - local: Local deployment (Ollama, etc.)
//   - custom: Custom OpenAI-compatible API
func Register() {
	// Register "Model" module.
	module.Use[
		*Model,
		*Model,
		*Model,
		*service.Base[*Model, *Model, *Model]](
		module.NewWrapper[*Model, *Model, *Model]("/api/models", "id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
		consts.PHASE_CREATE_MANY,
		consts.PHASE_DELETE_MANY,
		consts.PHASE_UPDATE_MANY,
		consts.PHASE_PATCH_MANY,
	)

	// Register "Provider" module.
	module.Use[
		*Provider,
		*Provider,
		*Provider,
		*service.Base[*Provider, *Provider, *Provider]](
		module.NewWrapper[*Provider, *Provider, *Provider]("/api/providers", "id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
		consts.PHASE_CREATE_MANY,
		consts.PHASE_DELETE_MANY,
		consts.PHASE_UPDATE_MANY,
		consts.PHASE_PATCH_MANY,
	)

	// Register "TestConnection" module.
	// Route: POST /api/ai/providers/test-connection
	// Request body: Provider (with config information)
	module.Use[
		*model.Empty,
		*modelaichat.Provider,
		*modelaichat.TestConnectionRsp,
		*serviceaichat.TestConnection](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.Provider,
			*modelaichat.TestConnectionRsp](
			"/ai/providers/test-connection",
			"id",
			false,
		),
		consts.PHASE_CREATE,
	)

	// Register "ListModels" module.
	// Route: POST /api/ai/providers/models
	// Request body: Provider (with config information)
	module.Use[
		*model.Empty,
		*modelaichat.Provider,
		*modelaichat.ListModelsRsp,
		*serviceaichat.ListModels](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.Provider,
			*modelaichat.ListModelsRsp](
			"/ai/providers/models",
			"id",
			false,
		),
		consts.PHASE_CREATE,
	)

	// Register "Chat" module.
	module.Use[
		*Chat,
		*Chat,
		*Chat,
		*service.Base[*Chat, *Chat, *Chat]](
		module.NewWrapper[*Chat, *Chat, *Chat]("/ai/chats", "chat_id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
		consts.PHASE_CREATE_MANY,
		consts.PHASE_DELETE_MANY,
		consts.PHASE_UPDATE_MANY,
		consts.PHASE_PATCH_MANY,
	)

	// Register "Message" module.
	module.Use[
		*Message,
		*Message,
		*Message,
		*service.Base[*Message, *Message, *Message]](
		module.NewWrapper[*Message, *Message, *Message]("/ai/chats/:chat_id/messages", "msg_id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
		consts.PHASE_CREATE_MANY,
		consts.PHASE_DELETE_MANY,
		consts.PHASE_UPDATE_MANY,
		consts.PHASE_PATCH_MANY,
	)

	// Register "MessageFeedback" module.
	module.Use[
		*MessageFeedBack,
		*MessageFeedBack,
		*MessageFeedBack,
		*service.Base[*MessageFeedBack, *MessageFeedBack, *MessageFeedBack]](
		module.NewWrapper[*MessageFeedBack, *MessageFeedBack, *MessageFeedBack]("/ai/chats/:chat_id/messages/:msg_id/feedback", "id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)
}
