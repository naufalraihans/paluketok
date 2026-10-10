How To Use Jev AI : 

Base URL : https://api.experientiallabs.ai/v1/systemone
API Key : <JEV_API_KEY> (simpan di server/.env, jangan di-commit)

Request : 
```
{
  "model": "jev-latest",
  "state": {
    "task": "Review a proposed change before a human decides whether to merge it.",
    "diff_summary": "Reject an empty project name before writing a record.",
    "test_summary": "The new empty-name test and existing creation tests pass. No integration tests were run."
  },
  "questions": {
    "review": {
      "type": "choice",
      "instructions": "Does the supplied evidence support accepting this change, or does it need further review? Treat missing evidence as a reason to review.",
      "criteria": {
        "accept": "The described change meets the task and the supplied tests cover its relevant behavior.",
        "review": "Evidence is missing, the tests are insufficient, or the change may be incorrect."
      }
    }
  }
}
```

Response : 
```
{
    "id": "decision_74b9c15110c49c94c09d960ed2a0144e",
    "model": "jev-latest",
    "answers": {
        "review": {
            "type": "choice",
            "choice": "review",
            "confidence": 0.98,
            "probabilities": {
                "accept": 0.01,
                "review": 0.99
            }
        }
    },
    "usage": {
        "input_tokens": 416,
        "output_tokens": 31,
        "cost": 0.0,
        "is_byok": false
    },
    "provider": "typesafe"
}
```




Ini arahan dari yg punya Jev AI : 
```
Add jev-latest (TypeSafe Jev) as a decision helper/tool for my coding agent.
Keep my coding agent on its current chat model. Jev is not a primary coding/chat model and does not generate chat completions.

Optional official skill setup (ask before installing; choose one route):
For Claude Code:
claude plugin marketplace add typesafe-ai/skills
claude plugin install typesafe@typesafe-ai
Then invoke /typesafe:typesafe-ai.
For other supported agents:
npx skills add typesafe-ai/skills --skill typesafe-ai
Then ask the agent to use the TypeSafe skill.
Skill reference: https://docs.typesafe.ai/agent-skill

The vendor skill/examples assume a direct TypeSafe key and origin. Override both for this integration:
- POST only to https://api.experientiallabs.ai/v1/systemone.
- Read the Bearer key only from JEV_API_KEY (sama dengan env MCP jev). If missing, stop and ask me to set it locally; never ask me to paste or log it.
- Do not request a TypeSafe API key or call the provider's endpoint. If the skill cannot use the gateway, write a small native HTTP helper instead.
- Send model "jev-latest" exactly, preserving any selected :free spelling.

Implement a native JSON request with model, state (text, object, or array), and questions (a map keyed by my question IDs). Each question needs type and instructions:
- choice: criteria maps options such as accept/review to their descriptions.
- noul: judge a proposition; optional criteria describe true and false.
- score: criteria is an ordered list of level descriptions.
Use a concise diff and test summary as state, not the entire repository. Ask independent questions; an answer is not fed into another question in the same request.
Approximately 32,000 input tokens shared by state and all question definitions; not a total context window or an output limit.
Gateway limits: at most 32 questions per request, 64 options per choice, and 2-10 levels per score. These are platform admission bounds, not provider limits.
Native request reference: https://docs.typesafe.ai/introduction/quickstart
Input-budget source: https://docs.typesafe.ai/primitives

Read answers by question ID (for example answers.review), and show the decision and reported usage to me. Probabilities and confidence are not guarantees of correctness. Never execute a returned choice, merge a change, or bypass review automatically.
Use a bounded timeout, surface HTTP and transport errors, and do not add automatic retries or Idempotency-Key: an unknown outcome may already be charged. This endpoint does not stream and has no Chat Completions, Responses, or Messages facade.
Prepare the helper and a mocked test first. Ask before making a potentially charged live call; do not claim it worked without a real response. Tell me which files you changed.

```