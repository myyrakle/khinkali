# khinkali

Khinkali는 OpenAPI 2.0 또는 3.x 문서를 바이트 배열이나 URL에서 읽고,
Swagger/OpenAPI 2.0을 OpenAPI 3.0.3으로 변환한 뒤 JSON 또는 YAML로
직렬화할 수 있는 Go 구현체입니다.

## 기능

- Swagger/OpenAPI 2.0 문서를 OpenAPI 3.0.3으로 변환합니다.
- JSON과 YAML 입력을 모두 받을 수 있습니다.
- 메모리에 있는 바이트 배열 또는 URL에서 문서를 읽을 수 있습니다.
- HTTP 서버를 띄우지 않고 일반 Go 패키지처럼 함수 호출로 사용할 수 있습니다.
- 필요하면 내장된 `GET /convert`, `POST /convert` HTTP API를 실행할 수 있습니다.
- 기본 출력은 JSON이며, 요청에 따라 YAML로 직렬화할 수 있습니다.

## 설치

다른 Go 모듈에서 사용할 때:

```sh
go get github.com/myyrakle/khinkali
```

이 저장소 안에서 개발할 때:

```sh
go mod tidy
go test ./...
```

## 라이브러리로 사용하기

로컬 Swagger/OpenAPI 파일을 변환하고 YAML로 출력하는 예시:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/myyrakle/khinkali/converter"
)

func main() {
	input, err := os.ReadFile("swagger.yaml")
	if err != nil {
		panic(err)
	}

	c := converter.New()
	doc, messages := c.ConvertContents(context.Background(), input)
	if len(messages) != 0 {
		panic(messages)
	}

	out, mediaType, err := converter.Encode(doc, converter.MediaTypeYAML)
	if err != nil {
		panic(err)
	}

	fmt.Println(mediaType)
	fmt.Print(string(out))
}
```

URL에 있는 문서를 변환하는 예시:

```go
package main

import (
	"context"
	"fmt"

	"github.com/myyrakle/khinkali/converter"
)

func main() {
	doc, messages := converter.New().ConvertLocation(
		context.Background(),
		"https://example.com/swagger.yaml",
	)
	if len(messages) != 0 {
		panic(messages)
	}

	out, _, err := converter.Encode(doc, converter.MediaTypeJSON)
	if err != nil {
		panic(err)
	}

	fmt.Print(string(out))
}
```

## 라이브러리 API

`converter.New() *converter.Converter`

기본 HTTP 클라이언트를 가진 변환기를 만듭니다. 기본 타임아웃은 30초입니다.

`(*Converter).ConvertContents(ctx context.Context, input []byte) (*openapi3.T, []string)`

이미 메모리에 올라와 있는 JSON 또는 YAML 문서를 변환합니다. 성공하면
`messages`는 빈 슬라이스입니다. 실패하면 `doc`은 `nil`이고 `messages`에
파싱 또는 변환 오류가 들어갑니다.

`(*Converter).ConvertLocation(ctx context.Context, rawURL string) (*openapi3.T, []string)`

`rawURL`에서 문서를 읽은 뒤 변환합니다. 가능한 경우 상대 외부 참조는 원본
URL을 기준으로 해석합니다.

`converter.Encode(doc *openapi3.T, mediaType string) ([]byte, string, error)`

변환된 문서를 직렬화합니다. 보기 좋은 JSON이 필요하면
`converter.MediaTypeJSON`, YAML이 필요하면 `converter.MediaTypeYAML`을
사용합니다.

`converter.PreferredMediaType(accept string) string`

원본 Swagger Converter와 같은 규칙으로 출력 미디어 타입을 고릅니다. 기본은
JSON이며, `application/yaml`은 허용되고 `application/json`은 허용되지 않을
때만 YAML을 선택합니다.

## HTTP 클라이언트 커스터마이징

`ConvertLocation`에서 사용하는 HTTP 클라이언트를 바꿀 수 있습니다:

```go
c := converter.New()
c.HTTPClient = &http.Client{
	Timeout: 10 * time.Second,
}
```

## HTTP 서버 실행

```sh
go run ./cmd/khinkali
```

기본 포트는 `:8080`입니다. `ADDR` 환경 변수로 바꿀 수 있습니다:

```sh
ADDR=:8081 go run ./cmd/khinkali
```

## HTTP API

요청 본문을 변환:

```sh
curl -X POST http://localhost:8080/convert \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/yaml' \
  --data-binary @swagger.json
```

원격 문서를 변환:

```sh
curl 'http://localhost:8080/convert?url=https://example.com/swagger.yaml'
```

JSON 출력 요청:

```sh
curl -X POST http://localhost:8080/convert \
  -H 'Accept: application/json' \
  --data-binary @swagger.yaml
```

YAML 출력 요청:

```sh
curl -X POST http://localhost:8080/convert \
  -H 'Accept: application/yaml' \
  --data-binary @swagger.yaml
```

## 오류 처리

라이브러리는 원본 Swagger Converter 컨트롤러와 비슷하게 변환 오류를
`[]string`으로 반환합니다. HTTP API는 문서가 없거나, 유효하지 않거나,
지원하지 않는 버전이면 `400 Bad Request`를 반환합니다.

`POST /convert` 요청 본문 크기 제한은 16 MiB입니다.
