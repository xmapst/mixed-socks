// Package docs GENERATED BY SWAG; DO NOT EDIT
// This file was generated by swaggo/swag
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {
            "name": "API Support",
            "url": "http://www.swagger.io/support",
            "email": "support@swagger.io"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/api/auth": {
            "get": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "get user auth state",
                "tags": [
                    "User"
                ],
                "summary": "Auth",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "enable user auth",
                "tags": [
                    "User"
                ],
                "summary": "Auth",
                "parameters": [
                    {
                        "description": "user auth",
                        "name": "auth",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/service.Auth"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            }
        },
        "/api/config": {
            "get": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "get mixed socks config",
                "tags": [
                    "Config"
                ],
                "summary": "get",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "update mixed socks config",
                "tags": [
                    "Config"
                ],
                "summary": "save",
                "parameters": [
                    {
                        "description": "config",
                        "name": "scripts",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/service.Conf"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            }
        },
        "/api/reload": {
            "post": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "reload mixed socks config",
                "tags": [
                    "Operate"
                ],
                "summary": "reload",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            }
        },
        "/api/start": {
            "post": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "start mixed socks",
                "tags": [
                    "Operate"
                ],
                "summary": "start",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            }
        },
        "/api/state": {
            "get": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "get mixed socks state",
                "tags": [
                    "Operate"
                ],
                "summary": "state",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            }
        },
        "/api/stop": {
            "post": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "stop mixed socks",
                "tags": [
                    "Operate"
                ],
                "summary": "stop",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            }
        },
        "/api/user": {
            "get": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "List all user",
                "tags": [
                    "User"
                ],
                "summary": "list",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            },
            "post": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "create or update user",
                "tags": [
                    "User"
                ],
                "summary": "save",
                "parameters": [
                    {
                        "description": "user",
                        "name": "user",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/service.User"
                        }
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            }
        },
        "/api/user/{username}": {
            "delete": {
                "security": [
                    {
                        "BasicAuth": []
                    }
                ],
                "description": "delete user",
                "tags": [
                    "User"
                ],
                "summary": "delete",
                "parameters": [
                    {
                        "type": "string",
                        "description": "username",
                        "name": "username",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            }
        },
        "/version": {
            "get": {
                "description": "当前服务器版本",
                "tags": [
                    "Version"
                ],
                "summary": "Version",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/api.Info"
                        }
                    },
                    "500": {
                        "description": "Internal Server Error",
                        "schema": {
                            "$ref": "#/definitions/api.JSONResult"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "api.Info": {
            "type": "object",
            "properties": {
                "buildTime": {
                    "type": "string"
                },
                "git": {
                    "type": "object",
                    "properties": {
                        "branch": {
                            "type": "string"
                        },
                        "commit": {
                            "type": "string"
                        },
                        "url": {
                            "type": "string"
                        }
                    }
                },
                "go": {
                    "type": "object",
                    "properties": {
                        "arch": {
                            "type": "string"
                        },
                        "os": {
                            "type": "string"
                        },
                        "version": {
                            "type": "string"
                        }
                    }
                },
                "user": {
                    "type": "object",
                    "properties": {
                        "email": {
                            "type": "string"
                        },
                        "name": {
                            "type": "string"
                        }
                    }
                },
                "version": {
                    "type": "string"
                }
            }
        },
        "api.JSONResult": {
            "type": "object",
            "properties": {
                "code": {
                    "type": "integer",
                    "example": 0
                },
                "data": {},
                "message": {
                    "type": "string",
                    "example": "消息"
                }
            }
        },
        "service.Auth": {
            "type": "object",
            "properties": {
                "enabled": {
                    "type": "boolean",
                    "example": false
                }
            }
        },
        "service.Conf": {
            "type": "object",
            "properties": {
                "cidr": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "example": [
                        "0.0.0.0/0"
                    ]
                },
                "host": {
                    "type": "string",
                    "example": "0.0.0.0"
                },
                "port": {
                    "type": "integer",
                    "example": 8090
                },
                "timeout": {
                    "type": "string",
                    "example": "30s"
                }
            }
        },
        "service.User": {
            "type": "object",
            "required": [
                "name"
            ],
            "properties": {
                "cidr": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "example": [
                        "0.0.0.0/0"
                    ]
                },
                "disabled": {
                    "type": "boolean",
                    "example": false
                },
                "name": {
                    "type": "string",
                    "example": "name"
                },
                "pass": {
                    "type": "string",
                    "example": "123456"
                },
                "remark": {
                    "type": "string",
                    "example": "小明"
                }
            }
        }
    },
    "securityDefinitions": {
        "BasicAuth": {
            "type": "basic"
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "v2.0.0",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "Mixed-Socks",
	Description:      "support socks4, socks5, http proxy all in one",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
