package springboot

import "strings"

type SecurityConfig struct {
	pg projectGenerator
}

func NewSecurityConfig(pg projectGenerator) *SecurityConfig {
	return &SecurityConfig{pg: pg}
}

func (s SecurityConfig) generate(pc ProjectConfig) error {
	if err := s.createSrcMainJava(pc); err != nil {
		return err
	}
	if err := s.createSrcTestJava(pc); err != nil {
		return err
	}
	return nil
}

func (s SecurityConfig) createSrcMainJava(pc ProjectConfig) error {
	if !pc.SecuritySupport && !pc.JwtSecuritySupport {
		return nil
	}
	basePackagePath := strings.ReplaceAll(pc.BasePackage, ".", "/")

	templateMap := map[string]string{}

	if pc.SecuritySupport || pc.JwtSecuritySupport {
		templateMap["users/Role.java.tmpl"] = "users/Role.java"
		templateMap["users/UserEntity.java.tmpl"] = "users/UserEntity.java"
		templateMap["users/UserDto.java.tmpl"] = "users/UserDto.java"
		templateMap["users/UserRepository.java.tmpl"] = "users/UserRepository.java"
		templateMap["users/UserService.java.tmpl"] = "users/UserService.java"
		templateMap["users/SecurityUser.java.tmpl"] = "users/SecurityUser.java"
		templateMap["config/SecurityConfig.java.tmpl"] = "config/SecurityConfig.java"
		templateMap["users/SecurityUserDetailsService.java.tmpl"] = "users/SecurityUserDetailsService.java"
		templateMap["users/UserContextUtils.java.tmpl"] = "users/UserContextUtils.java"
		templateMap["users/CreateUserCmd.java.tmpl"] = "users/CreateUserCmd.java"
	}

	if pc.SecuritySupport {
		templateMap["config/WebSecurityConfig.java.tmpl"] = "config/WebSecurityConfig.java"
		templateMap["config/WebAppExceptionHandler.java.tmpl"] = "config/GlobalExceptionHandler.java"
		templateMap["users/UserController.java.tmpl"] = "users/UserController.java"
	}

	if pc.JwtSecuritySupport {
		templateMap["config/JwtWebSecurityConfig.java.tmpl"] = "config/WebSecurityConfig.java"
		templateMap["users/AuthToken.java.tmpl"] = "users/AuthToken.java"
		templateMap["users/TokenHelper.java.tmpl"] = "users/TokenHelper.java"
		templateMap["users/JwtFilter.java.tmpl"] = "users/JwtFilter.java"
		templateMap["config/RestApiExceptionHandler.java.tmpl"] = "config/GlobalExceptionHandler.java"
		templateMap["users/LoginRestController.java.tmpl"] = "users/LoginRestController.java"
		templateMap["users/UserRestController.java.tmpl"] = "users/UserRestController.java"
	}

	for tmpl, filePath := range templateMap {
		err := s.pg.executeTemplate(pc, srcMainJavaPath+tmpl, srcMainJavaPath+basePackagePath+"/"+filePath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s SecurityConfig) createSrcTestJava(pc ProjectConfig) error {
	basePackagePath := strings.ReplaceAll(pc.BasePackage, ".", "/")

	templateMap := map[string]string{}

	if pc.SecuritySupport {
		templateMap["users/UserControllerTests.java.tmpl"] = "users/UserControllerTests.java"
	}

	if pc.JwtSecuritySupport {
		templateMap["users/LoginRestControllerTests.java.tmpl"] = "users/LoginRestControllerTests.java"
		templateMap["users/UserRestControllerTests.java.tmpl"] = "users/UserRestControllerTests.java"
	}

	for tmpl, filePath := range templateMap {
		err := s.pg.executeTemplate(pc, srcTestJavaPath+tmpl, srcTestJavaPath+basePackagePath+"/"+filePath)
		if err != nil {
			return err
		}
	}
	return nil
}
