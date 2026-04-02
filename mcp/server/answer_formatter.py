import os
import logging
import time

from openai import OpenAI

log = logging.getLogger("mcp-server")

ANSWER_PROMPT_FILE = os.path.join(os.path.dirname(__file__), "answer_prompt.txt")
with open(ANSWER_PROMPT_FILE, encoding="utf-8") as f:
    ANSWER_SYSTEM_PROMPT = f.read()


def format_answer(question: str, raw_json: str, openai_client: OpenAI, model: str) -> str:
    """Send the raw JSON query results back through OpenAI to produce a
    human-readable, natural-language answer."""
    log.info("Formatting answer via OpenAI (model=%s) …", model)
    t0 = time.perf_counter()
    response = openai_client.chat.completions.create(
        model=model,
        messages=[
            {"role": "system", "content": ANSWER_SYSTEM_PROMPT},
            {
                "role": "user",
                "content": f"Question: {question}\n\nData:\n{raw_json}",
            },
        ],
        temperature=0.3,
    )
    elapsed = time.perf_counter() - t0
    content = response.choices[0].message.content
    if content is None:
        raise ValueError("OpenAI returned an empty response — no answer was generated")
    answer = content.strip()
    log.info("Answer formatted in %.2fs", elapsed)
    return answer
