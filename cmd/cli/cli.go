package cli

import "strings"

// SplitArgsAndFlags separates positional arguments from flag arguments,
// allowing flags to appear anywhere in the argument list (before or after
// positional args).
//
// stringFlags is the set of flag names that take a string value (e.g. "out",
// "format"). Only these flags consume the next token as their value; all other
// flags are treated as boolean. Pass flag names without dashes: {"out", "format"}.
//
// Example:
//
//	SplitArgsAndFlags([]string{"llama3", "--out", "/tmp", "--verbose"}, map[string]bool{"out": true})
//	// positional: ["llama3"]
//	// flagArgs:   ["--out", "/tmp", "--verbose"]
func SplitArgsAndFlags(args []string, stringFlags map[string]bool) ([]string, []string) {
	// Separate flags from positional args before parsing so that flags may
	// appear anywhere in the argument list (e.g. after the model name).
	var positional []string
	var flagArgs []string

	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}

		flagArgs = append(flagArgs, a)

		// Skip inline-value flags (--out=/tmp): value is already part of the token.
		if strings.Contains(a, "=") {
			continue
		}

		// For string-value flags, consume the next token as the value only if
		// the flag name is in the known string-value set.
		name := flagName(a)
		if stringFlags[name] && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
			flagArgs = append(flagArgs, args[i])
		}
	}

	return positional, flagArgs
}

// flagName strips leading dashes from a flag token to get the bare name.
// "--out" → "out", "-v" → "v", "--out=/tmp" → "out" (though = case is
// handled before this is called).
func flagName(flag string) string {
	name := strings.TrimLeft(flag, "-")
	if idx := strings.Index(name, "="); idx != -1 {
		name = name[:idx]
	}
	return name

}
