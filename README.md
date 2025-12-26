# Yapp (Yet Another PDF Parser)

Because the world absolutely needed one more PDF parser — written in Go, caffeinated, and ready to turn stubborn PDFs into friendly Markdown.

## What is this?
- A Go-first take on parsing PDFs into structured Markdown for LLM usage.
- A playground for trying layout heuristics (headings, lists, tables, maybe even the occasional existential crisis).
- Borrowing battle scars from its two mates: the Go `pdf/` reader and the Python `pymupdf4llm/` Markdown extractor.
- Using a standard compiler approach: lexer -> parser -> AST -> Markdown

## Why Yapp?
- **Go-native**: no CPython hitchhikers or surprise virtualenvs.
- **Markdown-centric**: the goal is clean, LLM-ready text with structure, not just a word mess.
- **Hackable**: built to tinker with layout detection, fonts, and coordinates.
- **Soverighty**: to support off-cloud AI pipelines using Ollama LLMs for instance.

## Quick Start
```sh
# soon™
go run ./src/yapp --in sample.pdf --out sample.md
```

## Roadmap (a.k.a. TODO before we get distracted)
- Text extraction with font + position context.
- Heuristics for headings, paragraphs, lists, and tables.
- Image references (so your Markdown remembers the pretty pictures).

## Contributing
Issues and PRs welcome. Bad puns encouraged. Tests mandatory. Emojis optional.

## License
TBD, but expect something permissive and PDF-friendly.

Have fun!
/Bent
