# Gateway request shapes, per operation

Source: the gateway's request types (`app/service/gateway_*.go` in albitor) and
`docs/reference/gateway.md`. Names are case-sensitive and exactly as written here.

## The rule that applies to every operation

The gateway is **not** a pass-through to the AWS SDK. Each operation has an allow-list of fields.
The body must be a JSON object, a field not on the list is refused (`400 bad_request`) at any depth,
duplicate keys and trailing data are refused, and a value out of range is refused before anything is
charged. **Unknown fields are refused, never ignored. Never send AWS-SDK-only fields**: an SDK
request struct is the wrong thing to copy. Send exactly the fields below and no others.

What the gateway sets itself, so the app never names it: the Transcribe job name, the media
location, the output bucket, key and encryption, the data-access role, Transcribe's `Settings`
beyond the ones listed, Textract's output config, tokens and notification channel, Polly lexicons,
Translate terminologies, Bedrock guardrails and the model id, Location's intended use and tier.
An S3 URI, bucket, key, ARN or URL in any field is refused.

Wrong, from a real failure: Transcribe with the SDK's `TranscriptionJobName`, `MediaFormat`, `Media`
or `OutputBucketName` is `400` every time. Right: `UploadHandle` and a language.

**Body ceiling.** The whole request body, JSON and base64 included, is capped at **5,500,000 bytes**
by default (it was 1 MiB before albitor#4273), and larger is `413 body_too_large`. Base64 adds a
third, so inline raw bytes top out near 4 MB once the rest of the JSON is counted. Per operation:
every Location operation overrides it down to 4,096 bytes; no other operation overrides it. Send
bigger inputs through an upload slot (Transcribe, Textract async), never inline.

Reading a refusal: `*gateway.Error` carries `Field` and `Detail` when the gateway can say which
field it refused, so the text reads `gateway: 400 bad_request (field MediaFormat: unknown
field)`. This depends on the gateway change in albitor PR #4271; **until it is deployed, and on any
older gateway, `Field` and `Detail` are empty** and the text is a bare `400 bad_request`. Even then
only the Transcribe and Translate value checks name a field, plus unknown-field, duplicate-key and
wrong-scalar-type errors on every operation; any other failed check returns a bare 400. When the
text names no field, compare the request with the shape below.

"Req" below means required; "opt" means optional. `Idempotency-Key` is needed only where marked
(`gateway.WithIdempotencyKey(...)`); on any other operation do not send one.

## translate

`TranslateText` (metered per character)

| Field | | Rule |
| --- | --- | --- |
| `Text` | req | 1 to 10,000 bytes of UTF-8 |
| `SourceLanguageCode` | req | `en`, `pt`, `zh-TW`. **`auto` is refused**: name the language, or call Comprehend `DetectDominantLanguage` first |
| `TargetLanguageCode` | req | as above |
| `Settings` | opt | object: `Formality` (`FORMAL`\|`INFORMAL`), `Profanity` (`MASK`), `Brevity` (`ON`) |

No `TerminologyNames`, no `ParallelData`.

```json
{"Text": "Good morning", "SourceLanguageCode": "en", "TargetLanguageCode": "fr"}
```

Answer: `{TranslatedText, SourceLanguageCode, TargetLanguageCode, AppliedSettings?}`.

## transcribe

Three operations. The media travels through an upload slot, never in the request.

`CreateUploadSlot`: `{ContentType, MaxBytes}`, both req. `ContentType` is one of `audio/mpeg`,
`audio/mp4`, `audio/x-m4a`, `audio/m4a`, `audio/wav`, `audio/x-wav`, `audio/wave`, `audio/flac`,
`audio/x-flac`, `audio/ogg`, `audio/webm`, `audio/amr`, `video/mp4`, `video/webm`, `video/ogg`.
`MaxBytes` is 1 to 524,288,000. Use `svc.CreateUploadSlot` then `svc.Upload`; its answer is
`{UploadHandle, Url, Fields, ExpiresAt}`.

`StartTranscriptionJob` (**`Idempotency-Key` required**, metered per second of audio)

| Field | | Rule |
| --- | --- | --- |
| `UploadHandle` | req | the slot's `Handle`, alphanumeric, up to 64 characters |
| `LanguageCode` | one of | `en-GB`, `fr`: a code like `en` or `en-GB`. Exactly one of `LanguageCode` and `IdentifyLanguage` |
| `IdentifyLanguage` | one of | `true` to detect the language |
| `LanguageOptions` | opt | up to 30 language codes; only with `IdentifyLanguage: true` |
| `Settings` | opt | object, below |

`Settings` fields, each size only with its switch: `ShowSpeakerLabels` (bool) with
`MaxSpeakerLabels` (2 to 30; send it if and only if labels are on), `ChannelIdentification` (bool;
not together with speaker labels), `ShowAlternatives` (bool) with `MaxAlternatives` (2 to 10; if and
only if alternatives are on).

**Not fields**: `TranscriptionJobName` (the gateway names the job and returns the name), `Media`,
`MediaFileUri`, `MediaFormat`, `MediaSampleRateHertz`, `OutputBucketName`, `OutputKey`,
`OutputEncryptionKMSKeyId`, `DataAccessRoleArn`, `JobExecutionSettings`, `ContentRedaction`,
`Subtitles`, `ToxicityDetection`, `VocabularyName`, `ModelSettings`, `Tags`. The file's format is
read from its bytes.

```json
{"UploadHandle": "01J9ZQ8X2K4M6N7P8Q9R0S1T2V", "LanguageCode": "en-GB"}
```

```json
{"UploadHandle": "01J9ZQ8X2K4M6N7P8Q9R0S1T2V", "IdentifyLanguage": true,
 "LanguageOptions": ["en-GB", "fr"], "Settings": {"ShowSpeakerLabels": true, "MaxSpeakerLabels": 4}}
```

Answer: `{"TranscriptionJob": {"TranscriptionJobName": "<gateway's name>", "TranscriptionJobStatus":
"IN_PROGRESS", ...}}`. Keep the returned `TranscriptionJobName` to read the job.

`GetTranscriptionJob` (no key)

| Field | | Rule |
| --- | --- | --- |
| `TranscriptionJobName` | req | the name Start returned; an unknown or foreign name is `404 unknown_job` |
| `Inline` | opt | `true` returns the transcript JSON in `Result` when it is under 1 MiB; otherwise `Transcript.TranscriptFileUri` is a 5-minute URL: fetch it server-side |

```json
{"TranscriptionJobName": "<name returned by Start>", "Inline": true}
```

Poll until `TranscriptionJobStatus` is `COMPLETED` or `FAILED`. No `ListTranscriptionJobs`, no
streaming.

## comprehend

Declare the feature each operation needs (there is no default). No key. Text is 1 to 100,000 bytes
of UTF-8; a batch is 1 to 25 documents of 1 to 5,000 bytes. No `EndpointArn`, `Bytes` or
`DocumentReaderConfig`.

| Operation | Fields (all req) | Feature |
| --- | --- | --- |
| `DetectDominantLanguage` | `Text` | `language` |
| `DetectEntities` | `Text`, `LanguageCode` | `entities` |
| `DetectKeyPhrases` | `Text`, `LanguageCode` | `key-phrases` |
| `DetectSentiment` | `Text`, `LanguageCode` | `sentiment` |
| `DetectPiiEntities` | `Text`, `LanguageCode` | `pii` |
| `BatchDetectDominantLanguage` | `TextList` | `language` |
| `BatchDetectEntities`, `BatchDetectKeyPhrases`, `BatchDetectSentiment` | `TextList`, `LanguageCode` | `entities`, `key-phrases`, `sentiment` |

`LanguageCode` is `en`, `es`, `zh-TW`. `DetectPiiEntities` has no batch form.

```json
{"Text": "Amazon is based in Seattle.", "LanguageCode": "en"}
```

```json
{"TextList": ["First document", "Second document"], "LanguageCode": "en"}
```

A batch answers `{ResultList: [{Index, ...}], ErrorList: [{Index, ErrorCode}]}`.

## polly

`SynthesizeSpeech` (**`Idempotency-Key` required**, metered per character). The answer is the audio
bytes (`audio/mpeg` or `audio/ogg`), not JSON.

| Field | | Rule |
| --- | --- | --- |
| `Text` | req | up to 6,000 characters in all, up to 3,000 billed (SSML tags are not billed) |
| `VoiceId` | req | a voice name such as `Amy`, `Joanna` |
| `TextType` | opt | `text` (default) or `ssml`. SSML must be plain element tags: no CDATA, comments or processing instructions |
| `Engine` | opt | `standard` (default) or `neural`; declare the engine as the feature |
| `LanguageCode` | opt | the `xx-YY` form with a region: `en-GB`, `cmn-CN`; bare `en` is refused |
| `OutputFormat` | opt | `mp3` (default) or `ogg_vorbis`. No `pcm`, no `json` |
| `SampleRate` | opt | the string `"8000"`, `"16000"`, `"22050"` or `"24000"` (a string, not a number) |

Retrying with the same `Idempotency-Key` is `409 idempotency_replay_unavailable` (the audio is not
kept to replay): do not retry a Polly call with the same key, and use a fresh key for a deliberate
second call.

No `LexiconNames`, no `SpeechMarkTypes`. Long-form and generative voices are not offered.

```json
{"Text": "Your order has shipped.", "VoiceId": "Amy", "Engine": "neural", "OutputFormat": "mp3"}
```

## textract

Synchronous (one page, bytes in the request, no key, not idempotent so a retry is charged again):

- `DetectDocumentText`: `{"Document": {"Bytes": "<base64>"}}`. Go's `[]byte` marshals to base64.
- `AnalyzeDocument`: `{Document: {Bytes}, FeatureTypes, QueriesConfig?}`.

`Bytes` must be a single-page document (JPEG, PNG, one-page PDF or TIFF). Textract allows 10 MiB,
but the whole request body is capped at 5,500,000 bytes (see Body ceiling), so with base64 the raw
document must be under about 4 MB; larger is `413 body_too_large`. A bigger or multi-page document
goes through the asynchronous path below (upload slot, up to 500 MiB).

Asynchronous (a PDF or TIFF, through an upload slot; **starts need an `Idempotency-Key`**):

- `CreateUploadSlot`: `{ContentType, MaxBytes}`; `ContentType` is `application/pdf`, `image/tiff`,
  `image/jpeg` or `image/png`; `MaxBytes` up to 524,288,000.
- `StartDocumentTextDetection`: `{UploadHandle}`. Answer `{JobId}`.
- `StartDocumentAnalysis`: `{UploadHandle, FeatureTypes, QueriesConfig?}`. Answer `{JobId}`.
- `GetDocumentTextDetection`, `GetDocumentAnalysis` (no key): `{JobId, MaxResults?, NextToken?}`.
  `JobId` is the id Start returned; `MaxResults` is 0 to 1000; `NextToken` is the token the previous
  page returned. Answer `{JobStatus: "IN_PROGRESS"}`, `{JobStatus: "FAILED"}` or the page of blocks.

`FeatureTypes`: one to three of `TABLES`, `FORMS`, `QUERIES`, none repeated, each also declared as the
feature `tables`, `forms`, `queries`. `SIGNATURES` and `LAYOUT` are not offered. `QueriesConfig` is
`{"Queries": [{"Text", "Alias"?, "Pages"?}]}`, sent if and only if `QUERIES` is in `FeatureTypes`:
1 to 30 queries; `Text` 1 to 200 characters; `Alias` up to 100 characters of letters, digits, `_`,
`.`, space, `-`; `Pages` up to 5 entries like `"1"`, `"2-4"`, `"*"`.

**Not fields**: `S3Object`, `DocumentLocation`, `OutputConfig`, `KMSKeyId`, `NotificationChannel`,
`JobTag`, `ClientRequestToken`, `AdaptersConfig`, `HumanLoopConfig`.

```json
{"Document": {"Bytes": "<base64 of a one-page PNG>"}, "FeatureTypes": ["TABLES", "FORMS"]}
```

```json
{"UploadHandle": "01J9ZQ8X2K4M6N7P8Q9R0S1T2V", "FeatureTypes": ["QUERIES"],
 "QueriesConfig": {"Queries": [{"Text": "What is the invoice total?", "Alias": "total"}]}}
```

## bedrock

`Converse` (JSON answer) and `ConverseStream` (`application/x-ndjson`, one event per line). **No
`Idempotency-Key`** (a generation is not idempotent; send none). The body is AWS's Converse body in
**lowerCamelCase**, with the catalogue alias as `modelId`.

| Field | | Rule |
| --- | --- | --- |
| `modelId` | req | a catalogue alias you declared under `models` (`claude-haiku`, `claude-sonnet`, `claude-opus`, `nova-micro`, ...), never an AWS model id or ARN |
| `messages` | req | 1 to 256 turns: `{role: "user"\|"assistant", content: [block, ...]}`, 1 to 64 blocks each |
| `system` | opt | up to 16 blocks, each `{text}` or `{cachePoint}`, one of the two |
| `inferenceConfig` | opt | `{maxTokens?, temperature?, topP?, stopSequences?}`: `temperature` and `topP` 0 to 1, up to 4 stop sequences of 64 bytes. `maxTokens` above the model's cap is `400` (not clamped); unset means 4,096 |
| `toolConfig` | opt | `{tools: [{toolSpec: {name, description?, inputSchema: {json: {...}}}}], toolChoice?}`, 1 to 64 tools; `toolChoice` is exactly one of `{auto: {}}`, `{any: {}}`, `{tool: {name}}` |

A content block has exactly one of: `text`, `image` (`{format: png|jpeg|gif|webp, source}`),
`document` (`{format: pdf|txt|md|csv|html, name, source}`), `video` (`{format: mkv|mov|mp4|webm|flv|
mpeg|mpg|wmv|three_gp, source}`, only for a model that takes video, such as Nova Lite or Pro),
`toolUse` (`{toolUseId, name, input}`), `toolResult` (`{toolUseId, content: [{text}|{json}], status?}`),
`cachePoint` (`{type: "default"}`, Claude only). Counts are at most 1,024 content blocks in all, 20
images, 5 documents (a PDF up to 100 pages) and 1 video, and only for a model that takes them.

**Media `source` is exactly one of two things.** `{"bytes": "<base64>"}` is inline; or
`{"uploadHandle": "<handle>"}` names a file you uploaded first. A video takes the handle only.

- **Inline bytes** share the whole request body, which is capped at **5,500,000 bytes** (`413`
  above it), JSON and base64 framing included. That is about 4 MB of raw image or document in all
  the inline blocks together. Per item the caps are an image 3750 KiB and a document 4 MiB.
- **Anything bigger, and every video, goes by handle.** The handle names an object the gateway
  holds, so it adds almost nothing to the body. Get it with `CreateUploadSlot` on the `bedrock`
  service, then upload, then send `slot.Handle` as `uploadHandle`:

  ```go
  slot, err := svc.CreateUploadSlot(ctx, "bedrock", "video/mp4", int64(len(data))) // MaxBytes up to 1,000,000,000
  err = svc.Upload(ctx, slot, data)   // at once: the slot lasts 5 minutes
  // then Call(ctx, "bedrock", "Converse", req, &resp) with the handle in the block
  ```

  `ContentType` is one of `image/png|jpeg|gif|webp`, `application/pdf`,
  `text/plain|markdown|csv|html`, `video/mp4|quicktime|x-matroska|webm|x-flv|mpeg|x-ms-wmv|3gpp`.
  Upload before the `Converse` call and call at once; a handle that names nothing of yours (expired,
  another app's, not a handle) is `404 unknown_upload`. Never send an S3 URI, bucket or key:
  `source.s3Location` is `400`.

Exact blocks (one per media type; `bytes` is base64 of the file):

```json
{"image": {"format": "png", "source": {"bytes": "iVBORw0KGgo..."}}}
{"image": {"format": "jpeg", "source": {"uploadHandle": "01ARZ3NDEKTSV4RRFFQ69G5FAV"}}}
{"document": {"format": "pdf", "name": "invoice-2026-11", "source": {"uploadHandle": "01ARZ3NDEKTSV4RRFFQ69G5FAW"}}}
{"document": {"format": "txt", "name": "notes", "source": {"bytes": "SGVsbG8="}}}
{"video": {"format": "mp4", "source": {"uploadHandle": "01ARZ3NDEKTSV4RRFFQ69G5FAV"}}}
```

A whole request with a video:

```json
{"modelId": "nova-lite", "messages": [{"role": "user", "content": [
  {"text": "What happens in this clip?"},
  {"video": {"format": "mp4", "source": {"uploadHandle": "01ARZ3NDEKTSV4RRFFQ69G5FAV"}}}]}]}
```

The `400` names the field (`Field` on `*gateway.Error`):

| You sent | `Field` |
| --- | --- |
| both `bytes` and `uploadHandle`, or neither | `messages[].content[].<block>.source` (`image`, `document` or `video`), detail "exactly one of bytes and uploadHandle" |
| a video with `bytes` | `messages[].content[].video.source.bytes`: send a handle |
| `source.s3Location` | `source.s3Location` (unknown field) |

Also `400` before anything is billed: an image whose leading bytes are not its format, a PDF over
100 pages or not a PDF, an empty file, more than one video, or a media block for a model that does not
take it. A video reserves the model's whole input window against the app's balance up front (settled
from the tokens actually used), so a low balance can refuse it with `402`.

A document's `name` must match `^[A-Za-z0-9][A-Za-z0-9 ()\[\]-]{0,199}$`: letters, digits, spaces,
parentheses, square brackets and hyphens, starting with a letter or digit. **No dots, no
underscores**, so `invoice-2026-11` is valid and `invoice_2026.11.pdf` is `400`.

**Not fields**: `guardrailConfig`, `promptVariables`, `additionalModelRequestFields`,
`additionalModelResponseFieldPaths`, `performanceConfig`, `serviceTier`, `requestMetadata`,
an `s3Location` source, a cache `ttl`. Do not paste a Bedrock SDK input struct.

```json
{"modelId": "claude-haiku",
 "system": [{"text": "Answer in one sentence."}],
 "messages": [{"role": "user", "content": [{"text": "What is a hosted app?"}]}],
 "inferenceConfig": {"maxTokens": 300, "temperature": 0.2}}
```

## location

No `Idempotency-Key`. Declare `places` (the five place operations), `routes` (`CalculateRoutes`) or
`maps-third-party` (`GetTile`). A position is `[longitude, latitude]` (longitude first), longitude
-180 to 180, latitude -90 to 90. `Language` is a BCP 47 tag (`en`, `en-GB`). `IncludeCountries` is
up to 10 ISO alpha-3 codes (`GBR`). `MaxResults` is 0 to 20 (0 to 10 for `Autocomplete`).

| Operation | Fields |
| --- | --- |
| `Geocode`, `SearchText` | `QueryText` req (1 to 200 bytes); opt `BiasPosition`, `IncludeCountries`, `Language`, `MaxResults` |
| `Autocomplete` | as `Geocode`, plus opt `AdditionalFeatures` (only `["Core"]`) |
| `ReverseGeocode`, `SearchNearby` | `QueryPosition` req; opt `QueryRadius` (metres, 0 to 100,000), `Language`, `MaxResults` |
| `CalculateRoutes` | `Origin`, `Destination` req; opt `TravelMode` (`Car` only), `OptimizeRoutingFor` (`FastestRoute`\|`ShortestRoute`), `LegGeometryFormat` (`FlexiblePolyline`\|`Simple`) |
| `GetTile` | `Z`, `X`, `Y` req (integers; `Z` 0 to 22, `X` and `Y` below 2^Z); opt `Tileset` (`vector.basemap` only). Answers the tile's bytes |

No waypoints, alternatives, departure times, traffic, avoidances, `IntendedUse`, `AdditionalFeatures`
(other than Autocomplete's `Core`), `QueryId`, `NextToken` or filters.

```json
{"QueryText": "10 Downing Street, London", "IncludeCountries": ["GBR"], "MaxResults": 1}
```

```json
{"Origin": [-0.1276, 51.5072], "Destination": [-1.2577, 51.7520]}
```

## Answers that are not JSON

Polly, `ConverseStream` and `GetTile` answer audio, NDJSON or tile bytes. `svc.Call` decodes a JSON
answer into `resp`, so it fails on these: pass `nil` for `resp` only if the body is not needed,
otherwise add a small method to the app's own copy of the client that returns the raw body.
