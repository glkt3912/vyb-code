# 認知分析システムの技術手法 - わかりやすい解説

## はじめに

このドキュメントでは、vyb-codeプロジェクトで実装した認知分析システムの各技術手法について、専門用語を避けてわかりやすく解説します。

## 1. セマンティックエントロピー - 「意味の混乱度」を測る

### 基本的な考え方

**普通のエントロピー vs セマンティックエントロピー**

- **普通のエントロピー**: 「りんご」と「みかん」は違う単語なので多様性が高い
- **セマンティックエントロピー**: 「りんご」と「みかん」は両方とも「果物」なので意味的には近い

### 実際の例

```
質問: 「今日の天気はどうですか？」

応答A: 「晴れです」
応答B: 「快晴です」
応答C: 「雨が降っています」
```

**分析結果:**

- A と B は意味が似ている（同じグループ）
- C は意味が全く違う（別グループ）
- → 意味のバラツキが大きい = 不確実性が高い = 信頼度が低い

### 実装の仕組み

```go
// 1. 複数の応答を生成
responses := []string{"晴れです", "快晴です", "雨が降っています"}

// 2. 意味的に似ているものをグループ分け
group1 := []string{"晴れです", "快晴です"}      // 晴天グループ
group2 := []string{"雨が降っています"}           // 雨グループ

// 3. グループのバラツキを計算
// グループが多い = バラツキ大 = 信頼度低
// グループが少ない = バラツキ小 = 信頼度高
```

## 2. von Neumann エントロピー - 「情報のばらつき」を数学的に計算

### 量子物理学からのアイデア

**もともとの用途**: 量子の状態がどのくらい「あいまい」かを測る
**この実装での用途**: 言語の意味がどのくらい「あいまい」かを測る

### 簡単な例

```
レストランでの注文:
- 選択肢が2つ（カレー、ラーメン）→ 50%ずつの確率 → 適度な迷い
- 選択肢が10個 → 10%ずつの確率 → とても迷う（高エントロピー）
- 選択肢が1つだけ → 100%の確率 → 迷わない（低エントロピー）
```

### 実装での計算

```go
func calculateSimpleEntropy(groups []Group) float64 {
    total := 0
    for _, group := range groups {
        total += len(group.responses)
    }

    entropy := 0.0
    for _, group := range groups {
        // この群の確率 = この群の応答数 ÷ 全応答数
        probability := float64(len(group.responses)) / float64(total)

        // エントロピー計算（確率の対数を使った標準的な公式）
        if probability > 0 {
            entropy -= probability * math.Log2(probability)
        }
    }

    return entropy
}
```

## 3. 自然言語推論 (NLI) - 「文同士の論理関係」を判定

### 3つの基本関係

1. **含意 (Entailment)**: AならばBが成り立つ
   - A: 「太郎は犬を飼っている」
   - B: 「太郎はペットを飼っている」
   - → AならばBは必ず成り立つ

2. **矛盾 (Contradiction)**: AとBが同時に成り立たない
   - A: 「今日は晴れている」
   - B: 「今日は雨が降っている」
   - → 両方同時には起こらない

3. **中立 (Neutral)**: AとBに論理的関係がない
   - A: 「猫が好きです」
   - B: 「今日は火曜日です」
   - → 関係がない

### 実装での活用

```go
// 複数の応答間の関係を分析
responses := []string{
    "プロジェクトは成功しました",
    "タスクが完了しました",
    "失敗に終わりました"
}

// 関係を点数で表現
// 1.0 = 完全に含意, 0.0 = 完全に矛盾, 0.5 = 中立
matrix := [][]float64{
    {1.0, 0.8, 0.1},  // 「成功」と各応答の関係
    {0.8, 1.0, 0.2},  // 「完了」と各応答の関係
    {0.1, 0.2, 1.0},  // 「失敗」と各応答の関係
}

// この例では「失敗」だけが他と矛盾している
```

## 4. Guilford創造性理論 - 「創造性」を4つの角度で測定

### 4つの測定項目

1. **流暢性 (Fluency)**: アイデアの量
   - 「用途を考えてください」→ たくさん思いつく = 高得点

2. **柔軟性 (Flexibility)**: 発想の転換力
   - 「クリップの用途」→ 「書類留め、アクセサリー、工具」= 分野が多様 = 高得点

3. **独創性 (Originality)**: アイデアの珍しさ
   - 普通の答え（「書類留め」）= 低得点
   - 珍しい答え（「楽器のピック」）= 高得点

4. **精緻性 (Elaboration)**: アイデアの詳しさ
   - 簡単な説明 = 低得点
   - 詳しい説明 = 高得点

### 実装での測定方法

```go
func measureCreativity(response string) CreativityScore {
    // 1. 流暢性: アイデアの数を数える
    ideas := countIdeas(response)
    fluency := float64(ideas) / 10.0  // 10個を最大として正規化

    // 2. 柔軟性: 異なる分野のアイデアを数える
    categories := identifyCategories(response)
    flexibility := float64(len(categories)) / 5.0  // 5分野を最大として正規化

    // 3. 独創性: 一般的でない表現を探す
    uncommonWords := countUncommonWords(response)
    originality := float64(uncommonWords) / float64(len(allWords))

    // 4. 精緻性: 説明の詳しさを測る
    detailLevel := measureDetailLevel(response)
    elaboration := detailLevel / 100.0  // 100を最大として正規化

    return CreativityScore{
        Fluency: fluency,
        Flexibility: flexibility,
        Originality: originality,
        Elaboration: elaboration,
    }
}
```

## 5. Toulmin論証モデル - 「議論の構造」を分析

### 議論の6つの要素

1. **主張 (Claim)**: 「結論として何を言いたいか」
   - 例: 「このプロジェクトは成功する」

2. **根拠 (Ground)**: 「なぜそう言えるのか（事実）」
   - 例: 「予算内で進行している」

3. **論拠 (Warrant)**: 「根拠から主張への橋渡し」
   - 例: 「予算内進行は通常、成功の兆候である」

4. **後ろ盾 (Backing)**: 「論拠を支える理論や権威」
   - 例: 「過去の統計データによると...」

5. **限定詞 (Qualifier)**: 「どの程度確実か」
   - 例: 「おそらく」「確実に」

6. **反駁 (Rebuttal)**: 「例外や反対意見」
   - 例: 「ただし、技術的な問題が発生しなければ」

### 実装での活用

```go
func analyzeArgumentStructure(text string) ArgumentAnalysis {
    // テキストから各要素を自動抽出
    claims := findClaims(text)           // 「〜である」「〜すべき」
    grounds := findGrounds(text)         // 「なぜなら」「データによると」
    warrants := findWarrants(text)       // 「一般的に」「通常」
    qualifiers := findQualifiers(text)   // 「おそらく」「確実に」

    // 議論の完成度を計算
    completeness := float64(len(claims) + len(grounds) + len(warrants)) / 10.0

    return ArgumentAnalysis{
        Claims: claims,
        Grounds: grounds,
        ArgumentStrength: completeness,
    }
}
```

## 6. 概念ブレンディング - 「アイデアの組み合わせ」を分析

### 基本的な考え方

**普通の思考**: 1つの分野内で考える
**創造的思考**: 異なる分野のアイデアを組み合わせる

### 実例

```
質問: 「新しいアプリのアイデアを考えて」

普通の答え: 「SNSアプリ」（IT分野のみ）

創造的な答え: 「ガーデニング × ゲーム × SNS」
- ガーデニング: 植物を育てる
- ゲーム: レベルアップ、報酬
- SNS: 友達と共有
→ 「みんなで仮想植物を育てるゲーム」
```

### 実装での測定

```go
func measureConceptualBlending(response string) float64 {
    // 1. 応答に含まれる概念領域を特定
    domains := []string{}
    if containsIT(response) { domains = append(domains, "IT") }
    if containsNature(response) { domains = append(domains, "自然") }
    if containsArt(response) { domains = append(domains, "芸術") }
    // ... 他の分野も同様に

    // 2. 異なる分野の組み合わせを見つける
    blendingScore := 0.0
    for i := 0; i < len(domains); i++ {
        for j := i+1; j < len(domains); j++ {
            // 分野間の「距離」を計算
            distance := calculateDomainDistance(domains[i], domains[j])
            if distance > 0.5 { // 十分に異なる分野
                blendingScore += distance
            }
        }
    }

    return blendingScore
}
```

## 7. 認知負荷の計算 - 「頭の使用量」を測る

### 認知負荷に影響する要因

1. **不確実性**: わからないことが多い → 疲れる
2. **複雑さ**: 難しいことを考える → 疲れる
3. **時間**: 長時間考える → 疲れる
4. **作業記憶**: 同時にたくさん覚える → 疲れる

### 日常生活での例

```
低い認知負荷:
- 「1 + 1 = ?」
- すぐ答えがわかる、疲れない

高い認知負荷:
- 「17 × 23 を暗算で」
- 複数のステップ、時間がかかる、疲れる
```

### 実装での計算

```go
func calculateCognitiveLoad(
    confidence float64,    // 信頼度: 0.8 = かなり確信
    complexity float64,    // 複雑さ: 0.9 = とても複雑
    timeSpent time.Duration, // 処理時間: 5秒
) float64 {

    // 1. 不確実性による負荷（確信が低い = 負荷高）
    uncertaintyLoad := 1.0 - confidence  // 0.2

    // 2. 複雑性による負荷
    complexityLoad := complexity  // 0.9

    // 3. 時間による負荷（長時間 = 負荷高）
    timeLoad := float64(timeSpent.Seconds()) / 10.0  // 0.5

    // 4. 重み付き平均で統合
    totalLoad := uncertaintyLoad*0.4 + complexityLoad*0.3 + timeLoad*0.3
    // = 0.2*0.4 + 0.9*0.3 + 0.5*0.3 = 0.08 + 0.27 + 0.15 = 0.5

    return totalLoad  // 0.5 (中程度の認知負荷)
}
```

## 8. 実際のシステムでの統合

### 全体の流れ

```
1. ユーザーが質問
   ↓
2. システムが複数の応答を生成
   ↓
3. 各種分析を並行して実行:
   - セマンティックエントロピー → 信頼度
   - NLI分析 → 論理的一貫性
   - 創造性分析 → 創造性スコア
   - 論証分析 → 推論深度
   ↓
4. 結果を統合してスコア計算
   ↓
5. 最終的な認知パラメータを出力
```

### 統合計算の例

```go
func integrateAnalysisResults(
    confidence float64,    // 0.8 (セマンティックエントロピーから)
    creativity float64,    // 0.6 (創造性分析から)
    reasoning int,         // 5 (論証分析から)
) CognitiveResult {

    // 各スコアに重みをつけて統合
    overallQuality := confidence*0.4 + creativity*0.3 + float64(reasoning/10)*0.3
    // = 0.8*0.4 + 0.6*0.3 + 0.5*0.3 = 0.32 + 0.18 + 0.15 = 0.65

    // 処理戦略の決定
    var strategy string
    if confidence > 0.8 && reasoning > 6 {
        strategy = "高信頼度複雑処理"
    } else if creativity > 0.7 {
        strategy = "創造的探索"
    } else {
        strategy = "バランス型分析"
    }

    return CognitiveResult{
        Confidence: confidence,
        Creativity: creativity,
        ReasoningDepth: reasoning,
        Strategy: strategy,
        OverallQuality: overallQuality,
    }
}
```

## 9. 実用上のメリット

### 従来の問題

```go
// 固定値 - 入力に関係なく同じ
result := CognitiveResult{
    Confidence: 0.8,  // いつも同じ
    Creativity: 0.9,  // いつも同じ
    ReasoningDepth: 4, // いつも同じ
}
```

### 改良後の利点

```go
// 動的計算 - 入力に応じて変化
result := analyzeInput(userInput)
// 簡単な質問 → 高信頼度、低創造性、低推論深度
// 複雑な質問 → 低信頼度、高創造性、高推論深度
// 創造的な質問 → 中信頼度、高創造性、中推論深度
```

### 具体的な改善例

**質問1**: 「1+1は？」

- **従来**: 信頼度0.8, 創造性0.9, 推論深度4
- **改良後**: 信頼度0.95, 創造性0.1, 推論深度1

**質問2**: 「人工知能の未来について革新的なアイデアを」

- **従来**: 信頼度0.8, 創造性0.9, 推論深度4
- **改良後**: 信頼度0.6, 創造性0.8, 推論深度7

## 10. 技術的な工夫

### パフォーマンス最適化

1. **並列処理**: 複数の分析を同時実行
2. **キャッシュ**: 似たような質問の結果を保存
3. **適応的処理**: 簡単な質問は軽量処理、複雑な質問は詳細処理

### エラー処理

1. **フォールバック**: 一つの分析が失敗しても他でカバー
2. **段階的劣化**: 高精度が無理なら中精度、それも無理なら低精度
3. **自己診断**: システムが自分の信頼性を評価

## まとめ

これらの技術により、AIシステムの「考える力」を以下の観点で科学的に測定できるようになりました:

- **どのくらい確信しているか** (セマンティックエントロピー)
- **どのくらい創造的か** (Guilford理論)
- **どのくらい深く推論しているか** (Toulmin分析)
- **どのくらい頭を使っているか** (認知負荷)

これにより、AIの回答の質を客観的に評価し、より適切な支援を提供できるようになりました。
