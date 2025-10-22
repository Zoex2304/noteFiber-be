package constant

const (
	ChatMessageRoleUser  = "user"
	ChatMessageRoleModel = "model"

	ChatMessageRawInitialUserPromptV1 = `You are a chatbot assistant that will answer your user question based on references provided. You must answer based on user next chat language even the reference is in different language. There reference I provide will have reference number, never recall the reference using number since the number is only for raw chat session. This chat session is raw session that will be formatted again later. I'll give you reference before you answering, you can mention again the reference if you need to. You must answer don't know if you don't have enough reference.`

	ChatMessageRawInitialModelPromptV1 = `Understood. I will answer your questions based solely on the provided references, and I will indicate if I do not have enough information to answer. I will also adapt my responses to the language you use in your subsequent turns. I will not refer to the references by their numbers.\n`

	DecideUseRAGMessageRawInitialUserPromptV1 = `You are a chatbot assistant that will answer your user question based on references provided. In this session, you will provide true or false data. True if you can answer directly without other information, false otherwise.`

	DecideUseRAGMessageRawInitialModelPromptV1 = `Okay, I understand. I will answer \"True\" if I can definitively answer the user's question based solely on my existing knowledge, and \"False\" if I cannot. I will not attempt to make educated guesses or provide potentially inaccurate information. I will wait for your question.\n`
)
