package main

// Parser groups tokens into a simple AST, similar to yacc/bison phases.
type Parser struct {
	tokens []Token
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens}
}

func (p *Parser) Parse() DocumentNode {
	doc := DocumentNode{}
	var currentPage *PageNode
	var currentBlock *BlockNode
	var currentLine *LineNode
	newlineCount := 0

	flushLine := func() {
		if currentLine == nil || len(currentLine.Spans) == 0 {
			return
		}
		if currentBlock == nil {
			currentBlock = &BlockNode{}
		}
		currentBlock.Lines = append(currentBlock.Lines, *currentLine)
		currentLine = &LineNode{}
	}

	flushBlock := func() {
		if currentBlock == nil {
			return
		}
		flushLine()
		if len(currentBlock.Lines) == 0 || currentPage == nil {
			return
		}
		currentPage.Blocks = append(currentPage.Blocks, *currentBlock)
		currentBlock = &BlockNode{}
	}

	startPage := func(pageNum int) {
		if currentPage != nil {
			flushBlock()
			if len(currentPage.Blocks) > 0 {
				doc.Pages = append(doc.Pages, *currentPage)
			}
		}
		currentPage = &PageNode{Number: pageNum}
		currentBlock = &BlockNode{}
		currentLine = &LineNode{}
	}

	for _, tok := range p.tokens {
		if currentPage == nil {
			pageNum := tok.Pos.Page
			if pageNum == 0 {
				pageNum = 1
			}
			startPage(pageNum)
		}

		switch tok.Type {
		case TokenEOF:
			continue
		case TokenPageBreak:
			startPage(tok.Pos.Page)
		case TokenNewline:
			if currentLine != nil && len(currentLine.Spans) > 0 {
				flushLine()
				newlineCount++
			} else {
				newlineCount++
			}
			if newlineCount > 1 {
				flushBlock()
				newlineCount = 0
			}
		case TokenWord:
			newlineCount = 0
			currentLine.Spans = append(currentLine.Spans, TextSpan{
				Text: tok.Lexeme,
				Pos:  tok.Pos,
			})
		default:
			// ignore unknown tokens
		}
	}

	if currentPage != nil {
		flushBlock()
		if len(currentPage.Blocks) > 0 {
			doc.Pages = append(doc.Pages, *currentPage)
		}
	}

	return doc
}
