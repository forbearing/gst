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
	Conversation    = modelaichat.Conversation
	Message         = modelaichat.Message
	MessageFeedBack = modelaichat.MessageFeedback
	KnowledgeBase   = modelaichat.KnowledgeBase
	Document        = modelaichat.Document
	Chunk           = modelaichat.Chunk
	Prompt          = modelaichat.Prompt
	Agent           = modelaichat.Agent
	AgentTool       = modelaichat.AgentTool
	Favorite        = modelaichat.Favorite
)

// Register registers AI chat modules for managing AI providers, models, chats, messages, and knowledge bases.
//
// Models:
//   - Model: AI model configuration and metadata
//   - Provider: AI provider configuration (OpenAI, Anthropic, Ollama, etc.)
//   - Conversation: Conversation sessions
//   - Message: Messages within a chat conversation
//   - MessageFeedback: User feedback for messages
//   - KnowledgeBase: Knowledge bases for RAG (Retrieval-Augmented Generation)
//   - Document: Documents in knowledge bases
//   - Chunk: Text chunks from documents for vector search
//   - Prompt: Prompt templates for AI interactions
//   - Agent: AI agents with tools and RAG capabilities
//   - AgentTool: Tools/functions that agents can use
//   - Favorite: User's favorites (prompts, agents, etc.)
//
// Routes:
//
// Model module (full CRUD):
//   - POST     /api/ai/models
//   - DELETE   /api/ai/models/:id
//   - PUT      /api/ai/models/:id
//   - PATCH    /api/ai/models/:id
//   - GET      /api/ai/models
//   - GET      /api/ai/models/:id
//   - POST     /api/ai/models/batch
//   - DELETE   /api/ai/models/batch
//   - PUT      /api/ai/models/batch
//   - PATCH    /api/ai/models/batch
//
// Provider module (full CRUD):
//   - POST     /api/ai/providers
//   - DELETE   /api/ai/providers/:id
//   - PUT      /api/ai/providers/:id
//   - PATCH    /api/ai/providers/:id
//   - GET      /api/ai/providers
//   - GET      /api/ai/providers/:id
//   - POST     /api/ai/providers/batch
//   - DELETE   /api/ai/providers/batch
//   - PUT      /api/ai/providers/batch
//   - PATCH    /api/ai/providers/batch
//
// Conversation module (full CRUD):
//   - POST     /api/ai/conversations
//   - DELETE   /api/ai/conversations/:conv_id
//   - PUT      /api/ai/conversations/:conv_id
//   - PATCH    /api/ai/conversations/:conv_id
//   - GET      /api/ai/conversations
//   - GET      /api/ai/conversations/:conv_id
//   - POST     /api/ai/conversations/batch
//   - DELETE   /api/ai/conversations/batch
//   - PUT      /api/ai/conversations/batch
//   - PATCH    /api/ai/conversations/batch
//
// Message module (full CRUD):
//   - POST     /api/ai/conversations/:conv_id/messages
//   - DELETE   /api/ai/conversations/:conv_id/messages/:msg_id
//   - PUT      /api/ai/conversations/:conv_id/messages/:msg_id
//   - PATCH    /api/ai/conversations/:conv_id/messages/:msg_id
//   - GET      /api/ai/conversations/:conv_id/messages
//   - GET      /api/ai/conversations/:conv_id/messages/:msg_id
//   - POST     /api/ai/conversations/:conv_id/messages/batch
//   - DELETE   /api/ai/conversations/:conv_id/messages/batch
//   - PUT      /api/ai/conversations/:conv_id/messages/batch
//   - PATCH    /api/ai/conversations/:conv_id/messages/batch
//
// MessageFeedback module (full CRUD):
//   - POST     /api/ai/conversations/:conv_id/messages/:msg_id/feedback
//   - DELETE   /api/ai/conversations/:conv_id/messages/:msg_id/feedback/:id
//   - PUT      /api/ai/conversations/:conv_id/messages/:msg_id/feedback/:id
//   - PATCH    /api/ai/conversations/:conv_id/messages/:msg_id/feedback/:id
//   - GET      /api/ai/conversations/:conv_id/messages/:msg_id/feedback
//   - GET      /api/ai/conversations/:conv_id/messages/:msg_id/feedback/:id
//
// ProviderTestConn module:
//   - POST     /api/ai/providers/test-conn
//     Request body: Provider (with config information)
//     Response: ProviderTestConnRsp with success status and message
//
// ProviderSyncModels module:
//   - POST     /api/ai/providers/sync-models
//     Request body: Provider (with id)
//     Response: ProviderSyncModelsRsp with sync statistics
//
// KnowledgeBase module (full CRUD):
//   - POST     /api/ai/knowledge-bases
//   - DELETE   /api/ai/knowledge-bases/:kb_id
//   - PUT      /api/ai/knowledge-bases/:kb_id
//   - PATCH    /api/ai/knowledge-bases/:kb_id
//   - GET      /api/ai/knowledge-bases
//   - GET      /api/ai/knowledge-bases/:kb_id
//
// Document module (full CRUD):
//   - POST     /api/ai/knowledge-bases/:kb_id/documents
//   - DELETE   /api/ai/knowledge-bases/:kb_id/documents/:doc_id
//   - PUT      /api/ai/knowledge-bases/:kb_id/documents/:doc_id
//   - PATCH    /api/ai/knowledge-bases/:kb_id/documents/:doc_id
//   - GET      /api/ai/knowledge-bases/:kb_id/documents
//   - GET      /api/ai/knowledge-bases/:kb_id/documents/:doc_id
//
// Chunk module (full CRUD):
//   - POST     /api/ai/knowledge-bases/:kb_id/documents/:doc_id/chunks
//   - DELETE   /api/ai/knowledge-bases/:kb_id/documents/:doc_id/chunks/:chunk_id
//   - PUT      /api/ai/knowledge-bases/:kb_id/documents/:doc_id/chunks/:chunk_id
//   - PATCH    /api/ai/knowledge-bases/:kb_id/documents/:doc_id/chunks/:chunk_id
//   - GET      /api/ai/knowledge-bases/:kb_id/documents/:doc_id/chunks
//   - GET      /api/ai/knowledge-bases/:kb_id/documents/:doc_id/chunks/:chunk_id
//
// Prompt module (full CRUD):
//   - POST     /api/ai/prompts
//   - DELETE   /api/ai/prompts/:id
//   - PUT      /api/ai/prompts/:id
//   - PATCH    /api/ai/prompts/:id
//   - GET      /api/ai/prompts
//   - GET      /api/ai/prompts/:id
//   - POST     /api/ai/prompts/batch
//   - DELETE   /api/ai/prompts/batch
//   - PUT      /api/ai/prompts/batch
//   - PATCH    /api/ai/prompts/batch
//
// Agent module (full CRUD):
//   - POST     /api/ai/agents
//   - DELETE   /api/ai/agents/:id
//   - PUT      /api/ai/agents/:id
//   - PATCH    /api/ai/agents/:id
//   - GET      /api/ai/agents
//   - GET      /api/ai/agents/:id
//   - POST     /api/ai/agents/batch
//   - DELETE   /api/ai/agents/batch
//   - PUT      /api/ai/agents/batch
//   - PATCH    /api/ai/agents/batch
//
// AgentTool module (full CRUD):
//   - POST     /api/ai/tools
//   - DELETE   /api/ai/tools/:id
//   - PUT      /api/ai/tools/:id
//   - PATCH    /api/ai/tools/:id
//   - GET      /api/ai/tools
//   - GET      /api/ai/tools/:id
//   - POST     /api/ai/tools/batch
//   - DELETE   /api/ai/tools/batch
//   - PUT      /api/ai/tools/batch
//   - PATCH    /api/ai/tools/batch
//
// Favorite module (full CRUD):
//   - POST     /api/ai/favorites
//   - DELETE   /api/ai/favorites/:id
//   - PUT      /api/ai/favorites/:id
//   - PATCH    /api/ai/favorites/:id
//   - GET      /api/ai/favorites
//   - GET      /api/ai/favorites/:id
//   - POST     /api/ai/favorites/batch
//   - DELETE   /api/ai/favorites/batch
//   - PUT      /api/ai/favorites/batch
//   - PATCH    /api/ai/favorites/batch
//
// ChatCompletion module:
//   - POST     /api/ai/conversations/chat
//     Request body: ChatCompletionReq with model_id, messages, stream flag
//     Response: ChatCompletionRsp (for non-stream) or SSE stream (for stream)
//
// StopMessage module:
//   - POST     /api/ai/messages/stop
//     Request body: StopMessageReq with message_id
//     Response: Empty response
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
		module.NewWrapper[*Model, *Model, *Model]("/ai/models", "id", false),
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
		module.NewWrapper[*Provider, *Provider, *Provider]("/ai/providers", "id", false),
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

	// Register "ProviderSyncModels" module.
	// Route: POST /api/ai/providers/sync-models
	// Request body: Provider (with id)
	module.Use[
		*model.Empty,
		*modelaichat.Provider,
		*modelaichat.ProviderSyncModelsRsp,
		*serviceaichat.ProviderSyncModels](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.Provider,
			*modelaichat.ProviderSyncModelsRsp](
			"/ai/providers/sync-models",
			"id",
			false,
		),
		consts.PHASE_CREATE,
	)

	// Register "ProviderTestConn" module.
	// Route: POST /api/ai/providers/test-conn
	// Request body: Provider (with config information)
	module.Use[
		*model.Empty,
		*modelaichat.Provider,
		*modelaichat.ProviderTestConnRsp,
		*serviceaichat.ProviderTestConn](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.Provider,
			*modelaichat.ProviderTestConnRsp](
			"/ai/providers/test-conn",
			"id",
			false,
		),
		consts.PHASE_CREATE,
	)

	// Register "Conversation" module.
	module.Use[
		*Conversation,
		*Conversation,
		*Conversation,
		*service.Base[*Conversation, *Conversation, *Conversation]](
		module.NewWrapper[*Conversation, *Conversation, *Conversation]("/ai/conversations", "conv_id", false),
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
		module.NewWrapper[*Message, *Message, *Message]("/ai/conversations/:conv_id/messages", "msg_id", false),
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
		module.NewWrapper[*MessageFeedBack, *MessageFeedBack, *MessageFeedBack]("/ai/conversations/:conv_id/messages/:msg_id/feedback", "id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	// Register "KnowledgeBase" module.
	module.Use[
		*KnowledgeBase,
		*KnowledgeBase,
		*KnowledgeBase,
		*service.Base[*KnowledgeBase, *KnowledgeBase, *KnowledgeBase]](
		module.NewWrapper[*KnowledgeBase, *KnowledgeBase, *KnowledgeBase]("/ai/knowledge-bases", "kb_id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	// Register "Document" module.
	module.Use[
		*Document,
		*Document,
		*Document,
		*service.Base[*Document, *Document, *Document]](
		module.NewWrapper[*Document, *Document, *Document]("/ai/knowledge-bases/:kb_id/documents", "doc_id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	// Register "Chunk" module
	module.Use[
		*Chunk,
		*Chunk,
		*Chunk,
		*service.Base[*Chunk, *Chunk, *Chunk]](
		module.NewWrapper[*Chunk, *Chunk, *Chunk]("/ai/knowledge-bases/:kb_id/documents/:doc_id/chunks", "chunk_id", false),
		consts.PHASE_CREATE,
		consts.PHASE_DELETE,
		consts.PHASE_UPDATE,
		consts.PHASE_PATCH,
		consts.PHASE_LIST,
		consts.PHASE_GET,
	)

	// Register "Prompt" module
	module.Use[
		*Prompt,
		*Prompt,
		*Prompt,
		*service.Base[*Prompt, *Prompt, *Prompt]](
		module.NewWrapper[*Prompt, *Prompt, *Prompt]("/ai/prompts", "id", false),
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

	// Register "Agent" module
	module.Use[
		*Agent,
		*Agent,
		*Agent,
		*service.Base[*Agent, *Agent, *Agent]](
		module.NewWrapper[*Agent, *Agent, *Agent]("/ai/agents", "id", false),
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

	// Register "Favorite" module (for prompts, agents, etc.)
	module.Use[
		*Favorite,
		*Favorite,
		*Favorite,
		*service.Base[*Favorite, *Favorite, *Favorite]](
		module.NewWrapper[*Favorite, *Favorite, *Favorite]("/ai/favorites", "id", false),
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

	// Register "AgentTool" module
	module.Use[
		*AgentTool,
		*AgentTool,
		*AgentTool,
		*service.Base[*AgentTool, *AgentTool, *AgentTool]](
		module.NewWrapper[*AgentTool, *AgentTool, *AgentTool]("/ai/tools", "id", false),
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

	// Register "ChatCompletion" module
	//
	/*
		curl --location 'http://localhost:8090/api/ai/conversations/chat' \
		--header 'Content-Type: application/json' \
		--data '{
			"model_id": "claude-3-7-sonnet-20250219",
			"stream": true,
			"messages": [
				"你现在是一个精通 go 语言的专家",
				"简单介绍下 golang 这门语言,包括它的特性、用法和适用场景。"
			]

		}'
		curl --location 'http://localhost:8090/api/ai/conversations/chat' \
		--header 'Content-Type: application/json' \
		--data '{
			"model_id": "claude-3-7-sonnet-20250219",
			"stream": false,
			"messages": [
				"你现在是一个精通 go 语言的专家",
				"简单介绍下 golang 这门语言,包括它的特性、用法和适用场景。"
			]

		}'
	*/
	module.Use[
		*model.Empty,
		*modelaichat.ChatCompletionReq,
		*modelaichat.ChatCompletionRsp,
		*serviceaichat.ChatCompletion](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.ChatCompletionReq,
			*modelaichat.ChatCompletionRsp](
			"/ai/conversations/chat",
			"id",
			false,
		),
		consts.PHASE_CREATE,
	)

	// Register "StopMessage" module
	//
	/*
		curl --location --request POST 'http://localhost:8090/api/ai/messages/stop' \
		--header 'Content-Type: application/json' \
		--data '{
			"message_id": "xxxxx-message-id-xxxxx"
		}'
	*/
	module.Use[
		*model.Empty,
		*modelaichat.StopMessageReq,
		*modelaichat.StopMessageRsp,
		*serviceaichat.StopMessage](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.StopMessageReq,
			*modelaichat.StopMessageRsp](
			"/ai/messages/stop",
			"id",
			false,
		),
		consts.PHASE_CREATE,
	)

	// Register "RegenerateMessage" module
	//
	/*
		curl --location --request POST 'http://localhost:8090/api/ai/messages/regenerate' \
		--header 'Content-Type: application/json' \
		--data '{
			"message_id": "xxxxx-message-id-xxxxx",
			"stream": true
		}'
	*/
	module.Use[
		*model.Empty,
		*modelaichat.RegenerateMessageReq,
		*modelaichat.RegenerateMessageRsp,
		*serviceaichat.RegenerateMessage](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.RegenerateMessageReq,
			*modelaichat.RegenerateMessageRsp](
			"/ai/messages/regenerate",
			"id",
			false,
		),
		consts.PHASE_CREATE,
	)
}
