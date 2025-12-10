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
	Model             = modelaichat.Model
	Provider          = modelaichat.Provider
	Conversation      = modelaichat.Conversation
	Message           = modelaichat.Message
	MessageFeedBack   = modelaichat.MessageFeedback
	KnowledgeBase     = modelaichat.KnowledgeBase
	Document          = modelaichat.Document
	Chunk             = modelaichat.Chunk
	Prompt            = modelaichat.Prompt
	PromptFavorite    = modelaichat.PromptFavorite
	Agent             = modelaichat.Agent
	AgentTool         = modelaichat.AgentTool
	AgentFavorite     = modelaichat.AgentFavorite
	ChatCompletionReq = modelaichat.ChatCompletionReq
	ChatCompletionRsp = modelaichat.ChatCompletionRsp
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
//   - PromptFavorite: User's favorite prompts
//   - Agent: AI agents with tools and RAG capabilities
//   - AgentFavorite: User's favorite agents
//   - AgentTool: Tools/functions that agents can use
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
//   - DELETE   /api/ai/prompts/:pmt_id
//   - PUT      /api/ai/prompts/:pmt_id
//   - PATCH    /api/ai/prompts/:pmt_id
//   - GET      /api/ai/prompts
//   - GET      /api/ai/prompts/:pmt_id
//   - POST     /api/ai/prompts/batch
//   - DELETE   /api/ai/prompts/batch
//   - PUT      /api/ai/prompts/batch
//   - PATCH    /api/ai/prompts/batch
//
// PromptFavorite module (full CRUD):
//   - POST     /api/ai/prompts/:pmt_id/favorites
//   - DELETE   /api/ai/prompts/:pmt_id/favorites/:id
//   - PUT      /api/ai/prompts/:pmt_id/favorites/:id
//   - PATCH    /api/ai/prompts/:pmt_id/favorites/:id
//   - GET      /api/ai/prompts/:pmt_id/favorites
//   - GET      /api/ai/prompts/:pmt_id/favorites/:id
//   - POST     /api/ai/prompts/:pmt_id/favorites/batch
//   - DELETE   /api/ai/prompts/:pmt_id/favorites/batch
//   - PUT      /api/ai/prompts/:pmt_id/favorites/batch
//   - PATCH    /api/ai/prompts/:pmt_id/favorites/batch
//
// Agent module (full CRUD):
//   - POST     /api/ai/agents
//   - DELETE   /api/ai/agents/:agent_id
//   - PUT      /api/ai/agents/:agent_id
//   - PATCH    /api/ai/agents/:agent_id
//   - GET      /api/ai/agents
//   - GET      /api/ai/agents/:agent_id
//   - POST     /api/ai/agents/batch
//   - DELETE   /api/ai/agents/batch
//   - PUT      /api/ai/agents/batch
//   - PATCH    /api/ai/agents/batch
//
// AgentFavorite module (full CRUD):
//   - POST     /api/ai/agents/:agent_id/favorites
//   - DELETE   /api/ai/agents/:agent_id/favorites/:id
//   - PUT      /api/ai/agents/:agent_id/favorites/:id
//   - PATCH    /api/ai/agents/:agent_id/favorites/:id
//   - GET      /api/ai/agents/:agent_id/favorites
//   - GET      /api/ai/agents/:agent_id/favorites/:id
//   - POST     /api/ai/agents/:agent_id/favorites/batch
//   - DELETE   /api/ai/agents/:agent_id/favorites/batch
//   - PUT      /api/ai/agents/:agent_id/favorites/batch
//   - PATCH    /api/ai/agents/:agent_id/favorites/batch
//
// AgentTool module (full CRUD):
//   - POST     /api/ai/agents/tools
//   - DELETE   /api/ai/agents/tools/:id
//   - PUT      /api/ai/agents/tools/:id
//   - PATCH    /api/ai/agents/tools/:id
//   - GET      /api/ai/agents/tools
//   - GET      /api/ai/agents/tools/:id
//   - POST     /api/ai/agents/tools/batch
//   - DELETE   /api/ai/agents/tools/batch
//   - PUT      /api/ai/agents/tools/batch
//   - PATCH    /api/ai/agents/tools/batch
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
		*modelaichat.ProviderTestRsp,
		*serviceaichat.TestConnection](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.Provider,
			*modelaichat.ProviderTestRsp](
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
		*modelaichat.ProviderListModelsRsp,
		*serviceaichat.ListModels](
		module.NewWrapper[
			*model.Empty,
			*modelaichat.Provider,
			*modelaichat.ProviderListModelsRsp](
			"/ai/providers/models",
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
		module.NewWrapper[*KnowledgeBase, *KnowledgeBase, *KnowledgeBase]("/ai/rag/knowledge-bases", "kb_id", false),
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
		module.NewWrapper[*Document, *Document, *Document]("/ai/rag/knowledge-bases/:kb_id/documents", "doc_id", false),
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
		module.NewWrapper[*Chunk, *Chunk, *Chunk]("/ai/rag/knowledge-bases/:kb_id/documents/:doc_id/chunks", "id", false),
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
		module.NewWrapper[*Prompt, *Prompt, *Prompt]("/ai/prompts", "pmt_id", false),
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

	// Register "PromptFavorite" module
	module.Use[
		*PromptFavorite,
		*PromptFavorite,
		*PromptFavorite,
		*service.Base[*PromptFavorite, *PromptFavorite, *PromptFavorite]](
		module.NewWrapper[*PromptFavorite, *PromptFavorite, *PromptFavorite]("/ai/prompts/:pmt_id/favorites", "id", false),
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
		module.NewWrapper[*Agent, *Agent, *Agent]("/ai/agents", "agent_id", false),
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

	// Register "AgentFavorite" module
	module.Use[
		*AgentFavorite,
		*AgentFavorite,
		*AgentFavorite,
		*service.Base[*AgentFavorite, *AgentFavorite, *AgentFavorite]](
		module.NewWrapper[*AgentFavorite, *AgentFavorite, *AgentFavorite]("/ai/agents/:agent_id/favorites", "id", false),
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
		module.NewWrapper[*AgentTool, *AgentTool, *AgentTool]("/ai/agents/tools", "id", false),
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
		curl -X POST http://localhost:8080/api/ai/conversations/chat \
		  -H "Content-Type: application/json" \
		  -d '{
		    "model_id": "model-id",
		    "messages": ["你好"],
		    "stream": true
		  }'

		curl -X POST http://localhost:8080/api/ai/conversations/chat \
		  -H "Content-Type: application/json" \
		  -d '{
			"model_id": "model-id",
			"messages": ["你好"],
			"stream": false
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
}
