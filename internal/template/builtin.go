package template

// OfficialTemplates is the set of built-in template names that ship with the app.
// These templates cannot be deleted by the user.
var OfficialTemplates = map[string]bool{
	"gongwen":        true,
	"jiaoan-shicao":  true,
}

// IsOfficial returns true if the given template name is a built-in official template.
func IsOfficial(name string) bool {
	return OfficialTemplates[name]
}
