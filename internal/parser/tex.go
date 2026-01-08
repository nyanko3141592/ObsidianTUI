package parser

import (
	"regexp"
	"strings"
)

var (
	inlineMathPattern  = regexp.MustCompile(`\$([^$]+)\$`)
	displayMathPattern = regexp.MustCompile(`\$\$([^$]+)\$\$`)
)

var greekLetters = map[string]string{
	`\alpha`:   "α", `\beta`: "β", `\gamma`: "γ", `\delta`: "δ",
	`\epsilon`: "ε", `\zeta`: "ζ", `\eta`: "η", `\theta`: "θ",
	`\iota`:    "ι", `\kappa`: "κ", `\lambda`: "λ", `\mu`: "μ",
	`\nu`:      "ν", `\xi`: "ξ", `\pi`: "π", `\rho`: "ρ",
	`\sigma`:   "σ", `\tau`: "τ", `\upsilon`: "υ", `\phi`: "φ",
	`\chi`:     "χ", `\psi`: "ψ", `\omega`: "ω",
	`\Alpha`:   "Α", `\Beta`: "Β", `\Gamma`: "Γ", `\Delta`: "Δ",
	`\Epsilon`: "Ε", `\Zeta`: "Ζ", `\Eta`: "Η", `\Theta`: "Θ",
	`\Iota`:    "Ι", `\Kappa`: "Κ", `\Lambda`: "Λ", `\Mu`: "Μ",
	`\Nu`:      "Ν", `\Xi`: "Ξ", `\Pi`: "Π", `\Rho`: "Ρ",
	`\Sigma`:   "Σ", `\Tau`: "Τ", `\Upsilon`: "Υ", `\Phi`: "Φ",
	`\Chi`:     "Χ", `\Psi`: "Ψ", `\Omega`: "Ω",
}

var mathSymbols = map[string]string{
	`\infty`:     "∞", `\partial`: "∂", `\nabla`: "∇",
	`\sum`:       "Σ", `\prod`: "Π", `\int`: "∫",
	`\sqrt`:      "√", `\pm`: "±", `\mp`: "∓",
	`\times`:     "×", `\div`: "÷", `\cdot`: "·",
	`\leq`:       "≤", `\geq`: "≥", `\neq`: "≠",
	`\approx`:    "≈", `\equiv`: "≡", `\sim`: "∼",
	`\in`:        "∈", `\notin`: "∉", `\subset`: "⊂",
	`\supset`:    "⊃", `\subseteq`: "⊆", `\supseteq`: "⊇",
	`\cup`:       "∪", `\cap`: "∩", `\emptyset`: "∅",
	`\forall`:    "∀", `\exists`: "∃", `\neg`: "¬",
	`\land`:      "∧", `\lor`: "∨", `\implies`: "⟹",
	`\iff`:       "⟺", `\to`: "→", `\leftarrow`: "←",
	`\Rightarrow`: "⇒", `\Leftarrow`: "⇐",
	`\rightarrow`: "→", `\leftrightarrow`: "↔",
	`\uparrow`:   "↑", `\downarrow`: "↓",
	`\prime`:     "′", `\ldots`: "…", `\cdots`: "⋯",
	`\vdots`:     "⋮", `\ddots`: "⋱",
	`\angle`:     "∠", `\triangle`: "△", `\square`: "□",
	`\circ`:      "∘", `\bullet`: "•", `\star`: "★",
	`\hbar`:      "ℏ", `\ell`: "ℓ", `\Re`: "ℜ", `\Im`: "ℑ",
}

var superscripts = map[rune]rune{
	'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
	'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
	'+': '⁺', '-': '⁻', '=': '⁼', '(': '⁽', ')': '⁾',
	'n': 'ⁿ', 'i': 'ⁱ', 'x': 'ˣ', 'y': 'ʸ',
}

var subscripts = map[rune]rune{
	'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄',
	'5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
	'+': '₊', '-': '₋', '=': '₌', '(': '₍', ')': '₎',
	'a': 'ₐ', 'e': 'ₑ', 'i': 'ᵢ', 'j': 'ⱼ', 'n': 'ₙ',
	'o': 'ₒ', 'r': 'ᵣ', 'u': 'ᵤ', 'v': 'ᵥ', 'x': 'ₓ',
}

func RenderTeX(content string) string {
	content = displayMathPattern.ReplaceAllStringFunc(content, func(match string) string {
		inner := match[2 : len(match)-2]
		rendered := renderMathContent(inner)
		return "⟦ " + rendered + " ⟧"
	})

	content = inlineMathPattern.ReplaceAllStringFunc(content, func(match string) string {
		inner := match[1 : len(match)-1]
		return renderMathContent(inner)
	})

	return content
}

func renderMathContent(tex string) string {
	result := tex

	for cmd, sym := range greekLetters {
		result = strings.ReplaceAll(result, cmd, sym)
	}

	for cmd, sym := range mathSymbols {
		result = strings.ReplaceAll(result, cmd, sym)
	}

	result = renderSuperscripts(result)
	result = renderSubscripts(result)
	result = renderFractions(result)

	result = strings.ReplaceAll(result, `\{`, "{")
	result = strings.ReplaceAll(result, `\}`, "}")
	result = strings.ReplaceAll(result, `\ `, " ")
	result = strings.ReplaceAll(result, `\,`, " ")
	result = strings.ReplaceAll(result, `\;`, " ")
	result = strings.ReplaceAll(result, `\!`, "")

	return strings.TrimSpace(result)
}

func renderSuperscripts(s string) string {
	supPattern := regexp.MustCompile(`\^{([^}]+)}|\^(.)`)
	return supPattern.ReplaceAllStringFunc(s, func(match string) string {
		var content string
		if strings.HasPrefix(match, "^{") {
			content = match[2 : len(match)-1]
		} else {
			content = match[1:]
		}

		var result strings.Builder
		for _, r := range content {
			if sup, ok := superscripts[r]; ok {
				result.WriteRune(sup)
			} else {
				result.WriteRune(r)
			}
		}
		return result.String()
	})
}

func renderSubscripts(s string) string {
	subPattern := regexp.MustCompile(`_{([^}]+)}|_(.)`)
	return subPattern.ReplaceAllStringFunc(s, func(match string) string {
		var content string
		if strings.HasPrefix(match, "_{") {
			content = match[2 : len(match)-1]
		} else {
			content = match[1:]
		}

		var result strings.Builder
		for _, r := range content {
			if sub, ok := subscripts[r]; ok {
				result.WriteRune(sub)
			} else {
				result.WriteRune(r)
			}
		}
		return result.String()
	})
}

func renderFractions(s string) string {
	fracPattern := regexp.MustCompile(`\\frac{([^}]*)}{([^}]*)}`)
	return fracPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := fracPattern.FindStringSubmatch(match)
		if len(parts) == 3 {
			return "(" + parts[1] + "/" + parts[2] + ")"
		}
		return match
	})
}

func HasTeX(content string) bool {
	return inlineMathPattern.MatchString(content) || displayMathPattern.MatchString(content)
}
