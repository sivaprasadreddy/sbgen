package springboot

type ThymeleafConfig struct {
	pg projectGenerator
}

func NewThymeleafConfig(pg projectGenerator) *ThymeleafConfig {
	return &ThymeleafConfig{pg: pg}
}

func (t ThymeleafConfig) generate(pc ProjectConfig) error {
	if !pc.Thymeleaf {
		return nil
	}
	return t.createThymeleafTemplateFiles(pc)
}

func (t ThymeleafConfig) createThymeleafTemplateFiles(pc ProjectConfig) error {
	templateMap := map[string]string{}

	if pc.Thymeleaf {
		templateMap["static/css/styles.css"] = "static/css/styles.css"
		templateMap["templates/index.html.tmpl"] = "templates/index.html"
		templateMap["templates/layout.html.tmpl"] = "templates/layout.html"
		templateMap["templates/error/404.html.tmpl"] = "templates/error/404.html"
		templateMap["templates/error/500.html.tmpl"] = "templates/error/500.html"
	}

	if pc.Thymeleaf && pc.WebSecurity {
		templateMap["templates/login.html.tmpl"] = "templates/login.html"
		templateMap["templates/registration.html.tmpl"] = "templates/registration.html"
		templateMap["templates/error/403.html.tmpl"] = "templates/error/403.html"
	}

	for tmpl, filePath := range templateMap {
		err := t.pg.executeTemplate(pc, srcMainResourcesPath+tmpl, srcMainResourcesPath+filePath)
		if err != nil {
			return err
		}
	}
	return nil
}
