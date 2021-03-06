// Package syntaxhighlight provides syntax highlighting for code. It currently
// uses a language-independent lexer and performs decently on JavaScript, Java,
// Ruby, Python, Go, and C.
package halstead

import (
	"bytes"
	"io"
	"text/scanner"
	"text/template"
	"unicode"
	//"unicode/utf8"
	"github.com/sourcegraph/annotate"
)

// Kind represents a syntax highlighting kind (class) which will be assigned to tokens.
// A syntax highlighting scheme (style) maps text style properties to each token kind.
type Kind uint8
var totalOperators = 0
var differentOperators = 0
var totalOperands = 0
var differentOperands = 0
var comments = 0

var tokens []string
var operators map[string]int
var operands map[string]int

var wait_eq = false
var wait_eqq = false

var state = "contando"
var func_call = 0
var var_call = 0

var variable = ""

var i = 0

const (
	Whitespace Kind = iota
	String
	Keyword
	Comment
	Type
	Literal
	Punctuation
	Plaintext
	Tag
	HTMLTag
	HTMLAttrName
	HTMLAttrValue
	Decimal
)

//go:generate GoStringer -type=Kind

type Printer interface {
	Print(w io.Writer, kind Kind, tokText string) error
}

// HTMLConfig holds the HTML class configuration to be used by annotators when
// highlighting code.
type HTMLConfig struct {
	String        string
	Keyword       string
	Comment       string
	Type          string
	Literal       string
	Punctuation   string
	Plaintext     string
	Tag           string
	HTMLTag       string
	HTMLAttrName  string
	HTMLAttrValue string
	Decimal       string
	Whitespace    string
}

type HTMLPrinter HTMLConfig

// Class returns the set class for a given token Kind.
func (c HTMLConfig) Class(kind Kind,tokText string) string {
		i++
		tokens = append(tokens,tokText)
	
	switch kind {
	case String:
		func_call=0
		var_call=0
		variable = ""
	    operands[tokText]++
	    totalOperands = totalOperands + 1
	    return c.String
	case Keyword:
		func_call=0
		var_call=0
		variable = ""
	    operators[tokText]++
	    totalOperators = totalOperators + 1
	    return c.Keyword
	case Comment:
		func_call=0
		var_call=0
		variable = ""
	    comments++
	    return c.Comment
	case Type:
		if var_call == 2{
			var_call++
			operands[variable]--
			if operands[variable] == 0{
				delete(operands,variable)
			}
			operands[variable+"."+tokText]++
			variable = variable+"."+tokText
			break;
		}
		var_call++
		variable = tokText
	    operands[tokText]++
	    totalOperands = totalOperands + 1
	    return c.Type
	case Literal:
		func_call=0
		var_call=0
		variable = ""
	    return c.Literal
	case Punctuation:
		if tokText=="("&&(var_call==3||(var_call==1 && tokens[i-3] != "func")){
			operands[variable]--
			totalOperands--
			if operands[variable] == 0{
				delete(operands,variable)
			}
			operators[variable+"()"]++
			totalOperators++
		}
		if(tokText=="."){
			func_call++
			var_call++
		}else{
		    func_call=0
		    var_call=0
		    variable = ""
		}
	    if(tokText!="}" && tokText!="]" && tokText!=")" && tokText!="." && tokText!="{" && tokText!="("){
	        operators[tokText]++
	        totalOperators = totalOperators + 1
                if(tokText==":"){
               	    wait_eq = true
                }
                if(tokText =="=" && wait_eq){
            	    operators[":"]--
            	    operators["="]--
            	    character:=":"+"="
            	    operators[character]++
            	if operators[":"] == 0{
            	    delete(operators,":")
            	}
            	if operators["="] == 0{
            	    delete(operators,"=")
            	}
            	operators["declaracion"]++
            	totalOperators = totalOperators - 1
            	wait_eq = false
            }
            if(tokText=="!"){
               	wait_eqq = true
            }
            if(tokText =="=" && wait_eqq){
            	operators["!"]--
            	operators["="]--
            	operators["comparacion"]++
            	totalOperators = totalOperators - 1
            	wait_eqq = false
            }
        }
		return c.Punctuation
	case Plaintext:
		func_call++
		var_call++
		if var_call ==3{
			func_call = 0
			var_call = 0
			operands[variable]--
			if operands[variable] == 0{
				delete(operands,variable)
			}
			operands[variable+"."+tokText]++
		    variable = ""
			break
		}
		if var_call ==1{
			variable = tokText
		}
		operands[tokText]++
		totalOperands = totalOperands + 1
	    return c.Plaintext
	case Tag:
	    return c.Tag
	case HTMLTag:
		return c.HTMLTag
	case HTMLAttrName:
	    return c.HTMLAttrName
	case HTMLAttrValue:
	    return c.HTMLAttrValue
	case Decimal:
		operands[tokText]++
		totalOperands = totalOperands + 1
		return c.Decimal
	}
	i--
	tokens = tokens[:len(tokens)-1]
	return ""
}

func (p HTMLPrinter) Print(w io.Writer, kind Kind, tokText string) error {
	class := ((HTMLConfig)(p)).Class(kind,tokText)
	if class != "" {
		_, err := w.Write([]byte(`<span class="`))
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, class)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(`">`))
		if err != nil {
			return err
		}
	}
	template.HTMLEscape(w, []byte(tokText))
	if class != "" {
		_, err := w.Write([]byte(`</span>`))
		if err != nil {
			return err
		}
	}
	return nil
}

type Annotator interface {
	Annotate(start int, kind Kind, tokText string) (*annotate.Annotation, error)
}

type HTMLAnnotator HTMLConfig

func (a HTMLAnnotator) Annotate(start int, kind Kind, tokText string) (*annotate.Annotation, error) {
	class := ((HTMLConfig)(a)).Class(kind,tokText)
	if class != "" {
		left := []byte(`<span class="`)
		left = append(left, []byte(class)...)
		left = append(left, []byte(`">`)...)
		return &annotate.Annotation{
			Start: start, End: start + len(tokText),
			Left: left, Right: []byte("</span>"),
		}, nil
	}
	return nil, nil
}

// DefaultHTMLConfig's class names match those of google-code-prettify
// (https://code.google.com/p/google-code-prettify/).
var DefaultHTMLConfig = HTMLConfig{
	String:        "str",
	Keyword:       "kwd",
	Comment:       "com",
	Type:          "typ",
	Literal:       "lit",
	Punctuation:   "pun",
	Plaintext:     "pln",
	Tag:           "tag",
	HTMLTag:       "htm",
	HTMLAttrName:  "atn",
	HTMLAttrValue: "atv",
	Decimal:       "dec",
	Whitespace:    "",
}


func Print(s *scanner.Scanner, w io.Writer, p Printer) error {
	tok := s.Scan()
	operators = make(map[string]int)
    operands = make(map[string]int)
	for tok != scanner.EOF {
		tokText := s.TokenText()
		err := p.Print(w, tokenKind(tok, tokText), tokText)
		if err != nil {
			return err
		}

		tok = s.Scan()
	}
	return nil
}

func Annotate(src []byte, a Annotator) (annotate.Annotations, error) {
	s := NewScanner(src)

	var anns annotate.Annotations
	read := 0

	tok := s.Scan()
	for tok != scanner.EOF {
		tokText := s.TokenText()

		ann, err := a.Annotate(read, tokenKind(tok, tokText), tokText)
		if err != nil {
			return nil, err
		}
		read += len(tokText)
		if ann != nil {
			anns = append(anns, ann)
		}

		tok = s.Scan()
	}

	return anns, nil
}

func AsHTML(src []byte) (map[string]int,map[string]int,int,int, error) {
	totalOperators = 0
	differentOperators = 0
	totalOperands = 0
	var buf bytes.Buffer
	err := Print(NewScanner(src), &buf, HTMLPrinter(DefaultHTMLConfig))
	if err != nil {
		return operators,operands,0,0, err
	}

	differentOperators = len(operators)
	differentOperands = len(operands)
	return operators,operands,totalOperators,totalOperands, nil
}

// NewScanner is a helper that takes a []byte src, wraps it in a reader and creates a Scanner.
func NewScanner(src []byte) *scanner.Scanner {
	return NewScannerReader(bytes.NewReader(src))
}

// NewScannerReader takes a reader src and creates a Scanner.
func NewScannerReader(src io.Reader) *scanner.Scanner {
	var s scanner.Scanner
	s.Init(src)
	s.Error = func(_ *scanner.Scanner, _ string) {}
	s.Whitespace = 0
	s.Mode = s.Mode ^ scanner.SkipComments
	return &s
}

func tokenKind(tok rune, tokText string) Kind {
	switch tok {
	case scanner.Ident:
		if _, isKW := keywords[tokText]; isKW {
			return Keyword
		}else{
			return Type
		}
	case scanner.Float, scanner.Int:
		return Decimal
	case scanner.Char, scanner.String, scanner.RawString:
		return String
	case scanner.Comment:
		return Comment
	}
	if unicode.IsSpace(tok) {
		return Whitespace
	}
	return Punctuation
}
