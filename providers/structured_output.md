```go

func GenerateContent() {
    parameterMap := make(map[string]any)
    kwargs := map[string]any{"model": model, "contents": contents, "config": config}
    deepMarshal(kwargs, &parameterMap)
    ...
    schemaToVertex(parameterMap, ...)
}

func deepMarshal(input any, output *map[string]any) error {
    if inputBytes, err := json.Marshal(input); err != nil {
        return fmt.Errorf("deepMarshal: unable to marshal input: %w", err)
    } else if err := json.Unmarshal(inputBytes, output); err != nil {
        return fmt.Errorf("deepMarshal: unable to unmarshal input: %w", err)
    }
    return nil
}

func generateContentConfigToVertex(ac *apiClient, fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
toObject = make(map[string]any)

fromSystemInstruction := getValueByPath(fromObject, []string{"systemInstruction"})
if fromSystemInstruction != nil {
fromSystemInstruction, err = tContent(fromSystemInstruction)
if err != nil {
return nil, err
}

fromSystemInstruction, err = contentToVertex(fromSystemInstruction.(map[string]any), toObject)
if err != nil {
return nil, err
}

setValueByPath(parentObject, []string{"systemInstruction"}, fromSystemInstruction)
}

fromTemperature := getValueByPath(fromObject, []string{"temperature"})
if fromTemperature != nil {
setValueByPath(toObject, []string{"temperature"}, fromTemperature)
}

fromTopP := getValueByPath(fromObject, []string{"topP"})
if fromTopP != nil {
setValueByPath(toObject, []string{"topP"}, fromTopP)
}

fromTopK := getValueByPath(fromObject, []string{"topK"})
if fromTopK != nil {
setValueByPath(toObject, []string{"topK"}, fromTopK)
}

fromCandidateCount := getValueByPath(fromObject, []string{"candidateCount"})
if fromCandidateCount != nil {
setValueByPath(toObject, []string{"candidateCount"}, fromCandidateCount)
}

fromMaxOutputTokens := getValueByPath(fromObject, []string{"maxOutputTokens"})
if fromMaxOutputTokens != nil {
setValueByPath(toObject, []string{"maxOutputTokens"}, fromMaxOutputTokens)
}

fromStopSequences := getValueByPath(fromObject, []string{"stopSequences"})
if fromStopSequences != nil {
setValueByPath(toObject, []string{"stopSequences"}, fromStopSequences)
}

fromResponseLogprobs := getValueByPath(fromObject, []string{"responseLogprobs"})
if fromResponseLogprobs != nil {
setValueByPath(toObject, []string{"responseLogprobs"}, fromResponseLogprobs)
}

fromLogprobs := getValueByPath(fromObject, []string{"logprobs"})
if fromLogprobs != nil {
setValueByPath(toObject, []string{"logprobs"}, fromLogprobs)
}

fromPresencePenalty := getValueByPath(fromObject, []string{"presencePenalty"})
if fromPresencePenalty != nil {
setValueByPath(toObject, []string{"presencePenalty"}, fromPresencePenalty)
}

fromFrequencyPenalty := getValueByPath(fromObject, []string{"frequencyPenalty"})
if fromFrequencyPenalty != nil {
setValueByPath(toObject, []string{"frequencyPenalty"}, fromFrequencyPenalty)
}

fromSeed := getValueByPath(fromObject, []string{"seed"})
if fromSeed != nil {
setValueByPath(toObject, []string{"seed"}, fromSeed)
}

fromResponseMimeType := getValueByPath(fromObject, []string{"responseMimeType"})
if fromResponseMimeType != nil {
setValueByPath(toObject, []string{"responseMimeType"}, fromResponseMimeType)
}

fromResponseSchema := getValueByPath(fromObject, []string{"responseSchema"})
if fromResponseSchema != nil {
fromResponseSchema, err = tSchema(fromResponseSchema)
if err != nil {
return nil, err
}

fromResponseSchema, err = schemaToVertex(fromResponseSchema.(map[string]any), toObject)
if err != nil {
return nil, err
}

setValueByPath(toObject, []string{"responseSchema"}, fromResponseSchema)
}

fromResponseJsonSchema := getValueByPath(fromObject, []string{"responseJsonSchema"})
if fromResponseJsonSchema != nil {
setValueByPath(toObject, []string{"responseJsonSchema"}, fromResponseJsonSchema)
}

fromRoutingConfig := getValueByPath(fromObject, []string{"routingConfig"})
if fromRoutingConfig != nil {
setValueByPath(toObject, []string{"routingConfig"}, fromRoutingConfig)
}

fromModelSelectionConfig := getValueByPath(fromObject, []string{"modelSelectionConfig"})
if fromModelSelectionConfig != nil {
fromModelSelectionConfig, err = modelSelectionConfigToVertex(fromModelSelectionConfig.(map[string]any), toObject)
if err != nil {
return nil, err
}

setValueByPath(toObject, []string{"modelConfig"}, fromModelSelectionConfig)
}

fromSafetySettings := getValueByPath(fromObject, []string{"safetySettings"})
if fromSafetySettings != nil {
fromSafetySettings, err = applyConverterToSlice(fromSafetySettings.([]any), safetySettingToVertex)
if err != nil {
return nil, err
}

setValueByPath(parentObject, []string{"safetySettings"}, fromSafetySettings)
}

fromTools := getValueByPath(fromObject, []string{"tools"})
if fromTools != nil {
fromTools, err = applyItemTransformerToSlice(fromTools.([]any), tTool)
if err != nil {
return nil, err
}

fromTools, err = tTools(fromTools)
if err != nil {
return nil, err
}

fromTools, err = applyConverterToSlice(fromTools.([]any), toolToVertex)
if err != nil {
return nil, err
}

setValueByPath(parentObject, []string{"tools"}, fromTools)
}

fromToolConfig := getValueByPath(fromObject, []string{"toolConfig"})
if fromToolConfig != nil {
fromToolConfig, err = toolConfigToVertex(fromToolConfig.(map[string]any), toObject)
if err != nil {
return nil, err
}

setValueByPath(parentObject, []string{"toolConfig"}, fromToolConfig)
}

fromLabels := getValueByPath(fromObject, []string{"labels"})
if fromLabels != nil {
setValueByPath(parentObject, []string{"labels"}, fromLabels)
}

fromCachedContent := getValueByPath(fromObject, []string{"cachedContent"})
if fromCachedContent != nil {
fromCachedContent, err = tCachedContentName(ac, fromCachedContent)
if err != nil {
return nil, err
}

setValueByPath(parentObject, []string{"cachedContent"}, fromCachedContent)
}

fromResponseModalities := getValueByPath(fromObject, []string{"responseModalities"})
if fromResponseModalities != nil {
setValueByPath(toObject, []string{"responseModalities"}, fromResponseModalities)
}

fromMediaResolution := getValueByPath(fromObject, []string{"mediaResolution"})
if fromMediaResolution != nil {
setValueByPath(toObject, []string{"mediaResolution"}, fromMediaResolution)
}

fromSpeechConfig := getValueByPath(fromObject, []string{"speechConfig"})
if fromSpeechConfig != nil {
fromSpeechConfig, err = tSpeechConfig(fromSpeechConfig)
if err != nil {
return nil, err
}

fromSpeechConfig, err = speechConfigToVertex(fromSpeechConfig.(map[string]any), toObject)
if err != nil {
return nil, err
}

setValueByPath(toObject, []string{"speechConfig"}, fromSpeechConfig)
}

fromAudioTimestamp := getValueByPath(fromObject, []string{"audioTimestamp"})
if fromAudioTimestamp != nil {
setValueByPath(toObject, []string{"audioTimestamp"}, fromAudioTimestamp)
}

fromThinkingConfig := getValueByPath(fromObject, []string{"thinkingConfig"})
if fromThinkingConfig != nil {
fromThinkingConfig, err = thinkingConfigToVertex(fromThinkingConfig.(map[string]any), toObject)
if err != nil {
return nil, err
}

setValueByPath(toObject, []string{"thinkingConfig"}, fromThinkingConfig)
}

return toObject, nil
}


func schemaToVertex(fromObject map[string]any, parentObject map[string]any) (toObject map[string]any, err error) {
    toObject = make(map[string]any)
    
    fromAnyOf := getValueByPath(fromObject, []string{"anyOf"})
    if fromAnyOf != nil {
        setValueByPath(toObject, []string{"anyOf"}, fromAnyOf)
    }
    
    fromDefault := getValueByPath(fromObject, []string{"default"})
    if fromDefault != nil {
        setValueByPath(toObject, []string{"default"}, fromDefault)
    }
    
    fromDescription := getValueByPath(fromObject, []string{"description"})
    if fromDescription != nil {
        setValueByPath(toObject, []string{"description"}, fromDescription)
    }
    
    fromEnum := getValueByPath(fromObject, []string{"enum"})
    if fromEnum != nil {
        setValueByPath(toObject, []string{"enum"}, fromEnum)
    }
    
    fromExample := getValueByPath(fromObject, []string{"example"})
    if fromExample != nil {
        setValueByPath(toObject, []string{"example"}, fromExample)
    }
    
    fromFormat := getValueByPath(fromObject, []string{"format"})
    if fromFormat != nil {
        setValueByPath(toObject, []string{"format"}, fromFormat)
	}
    
    fromItems := getValueByPath(fromObject, []string{"items"})
    if fromItems != nil {
        setValueByPath(toObject, []string{"items"}, fromItems)
    }
    
    fromMaxItems := getValueByPath(fromObject, []string{"maxItems"})
    if fromMaxItems != nil {
        setValueByPath(toObject, []string{"maxItems"}, fromMaxItems)
	}
    
    fromMaxLength := getValueByPath(fromObject, []string{"maxLength"})
    if fromMaxLength != nil {
        setValueByPath(toObject, []string{"maxLength"}, fromMaxLength)
    }
    
    fromMaxProperties := getValueByPath(fromObject, []string{"maxProperties"})
    if fromMaxProperties != nil {
        setValueByPath(toObject, []string{"maxProperties"}, fromMaxProperties)
    }
    
    fromMaximum := getValueByPath(fromObject, []string{"maximum"})
    if fromMaximum != nil {
        setValueByPath(toObject, []string{"maximum"}, fromMaximum)
    }
    
    fromMinItems := getValueByPath(fromObject, []string{"minItems"})
    if fromMinItems != nil {
        setValueByPath(toObject, []string{"minItems"}, fromMinItems)
    }
    
    fromMinLength := getValueByPath(fromObject, []string{"minLength"})
    if fromMinLength != nil {
        setValueByPath(toObject, []string{"minLength"}, fromMinLength)
    }
    
    fromMinProperties := getValueByPath(fromObject, []string{"minProperties"})
    if fromMinProperties != nil {
        setValueByPath(toObject, []string{"minProperties"}, fromMinProperties)
    }
    
    fromMinimum := getValueByPath(fromObject, []string{"minimum"})
    if fromMinimum != nil {
        setValueByPath(toObject, []string{"minimum"}, fromMinimum)
    }
    
    fromNullable := getValueByPath(fromObject, []string{"nullable"})
    if fromNullable != nil {
        setValueByPath(toObject, []string{"nullable"}, fromNullable)
    }
    
    fromPattern := getValueByPath(fromObject, []string{"pattern"})
    if fromPattern != nil {
        setValueByPath(toObject, []string{"pattern"}, fromPattern)
    }
    
    fromProperties := getValueByPath(fromObject, []string{"properties"})
    if fromProperties != nil {
        setValueByPath(toObject, []string{"properties"}, fromProperties)
    }
    
    fromPropertyOrdering := getValueByPath(fromObject, []string{"propertyOrdering"})
    if fromPropertyOrdering != nil {
        setValueByPath(toObject, []string{"propertyOrdering"}, fromPropertyOrdering)
    }
    
    fromRequired := getValueByPath(fromObject, []string{"required"})
    if fromRequired != nil {
        setValueByPath(toObject, []string{"required"}, fromRequired)
    }
    
    fromTitle := getValueByPath(fromObject, []string{"title"})
    if fromTitle != nil {
        setValueByPath(toObject, []string{"title"}, fromTitle)
    }
    
    fromType := getValueByPath(fromObject, []string{"type"})
    if fromType != nil {
        setValueByPath(toObject, []string{"type"}, fromType)
    }
    
    return toObject, nil
}

// getValueByPath retrieves a value from a nested map or slice or struct based on a path of keys.
//
// Examples:
//
//	getValueByPath(map[string]any{"a": {"b": "v"}}, []string{"a", "b"})
//	  -> "v"
//	getValueByPath(map[string]any{"a": {"b": [{"c": "v1"}, {"c": "v2"}]}}, []string{"a", "b[]", "c"})
//	  -> []any{"v1", "v2"}
func getValueByPath(data any, keys []string) any {
    if len(keys) == 1 && keys[0] == "_self" {
        return data
    }
	
    if len(keys) == 0 {
        return nil
    }
	
    var current any = data
    for i, key := range keys {
        if strings.HasSuffix(key, "[]") {
            keyName := key[:len(key)-2]
            switch v := current.(type) {
            case map[string]any:
                if sliceData, ok := v[keyName]; ok {
                    var result []any
					
                    switch concreteSliceData := sliceData.(type) {
                    case []map[string]any:
                        for _, d := range concreteSliceData {
                            result = append(result, getValueByPath(d, keys[i+1:]))
                        }
                    case []any:
                        for _, d := range concreteSliceData {
                            result = append(result, getValueByPath(d, keys[i+1:]))
                        }
                    default:
                        return nil
                }
                    return result
                } else {
                    return nil
                }
            default:
                return nil
            }
        } else {
            switch v := current.(type) {
            case map[string]any:
                current = v[key]
            default:
                return nil
            }
        }
    }
    return current
}

```