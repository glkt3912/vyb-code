package ui

import "github.com/charmbracelet/lipgloss"

// カラーテーマ定義
type Theme struct {
	Name       string
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Error      lipgloss.Color
	Info       lipgloss.Color
	Background lipgloss.Color
	Foreground lipgloss.Color
	Border     lipgloss.Color
	Accent     lipgloss.Color
}

// テーマタイプ
type ThemeType string

const (
	ThemeDark  ThemeType = "dark"
	ThemeLight ThemeType = "light"
	ThemeAuto  ThemeType = "auto"
)

// ダークテーマ
func DarkTheme() Theme {
	return Theme{
		Name:       "dark",
		Primary:    lipgloss.Color("#00FFFF"), // シアン
		Secondary:  lipgloss.Color("#808080"), // グレー
		Success:    lipgloss.Color("#00FF00"), // 緑
		Warning:    lipgloss.Color("#FFFF00"), // 黄色
		Error:      lipgloss.Color("#FF0000"), // 赤
		Info:       lipgloss.Color("#0080FF"), // 青
		Background: lipgloss.Color("#1E1E1E"), // 濃いグレー
		Foreground: lipgloss.Color("#FFFFFF"), // 白
		Border:     lipgloss.Color("#444444"), // 中グレー
		Accent:     lipgloss.Color("#FF69B4"), // ピンク
	}
}

// ライトテーマ
func LightTheme() Theme {
	return Theme{
		Name:       "light",
		Primary:    lipgloss.Color("#0066CC"), // 濃い青
		Secondary:  lipgloss.Color("#666666"), // ダークグレー
		Success:    lipgloss.Color("#009900"), // 濃い緑
		Warning:    lipgloss.Color("#CC6600"), // オレンジ
		Error:      lipgloss.Color("#CC0000"), // 濃い赤
		Info:       lipgloss.Color("#0066FF"), // 青
		Background: lipgloss.Color("#FFFFFF"), // 白
		Foreground: lipgloss.Color("#000000"), // 黒
		Border:     lipgloss.Color("#CCCCCC"), // 薄いグレー
		Accent:     lipgloss.Color("#9966CC"), // 紫
	}
}

// vybテーマ（ブランド専用）
func VybTheme() Theme {
	return Theme{
		Name:       "vyb",
		Primary:    lipgloss.Color("#FF6B9D"), // vyb ピンク
		Secondary:  lipgloss.Color("#6BCF7F"), // vyb グリーン
		Success:    lipgloss.Color("#4ECDC4"), // vyb ティール
		Warning:    lipgloss.Color("#FFE66D"), // vyb イエロー
		Error:      lipgloss.Color("#FF6B6B"), // vyb レッド
		Info:       lipgloss.Color("#74B9FF"), // vyb ブルー
		Background: lipgloss.Color("#2D3748"), // vyb ダーク
		Foreground: lipgloss.Color("#EDF2F7"), // vyb ライト
		Border:     lipgloss.Color("#4A5568"), // vyb ボーダー
		Accent:     lipgloss.Color("#9F7AEA"), // vyb パープル
	}
}

// テーママネージャー
type ThemeManager struct {
	current    Theme
	available  map[ThemeType]Theme
	autoDetect bool
}

// 新しいテーママネージャーを作成
func NewThemeManager() *ThemeManager {
	tm := &ThemeManager{
		available:  make(map[ThemeType]Theme),
		autoDetect: true,
	}

	// 標準テーマを登録
	tm.available[ThemeDark] = DarkTheme()
	tm.available[ThemeLight] = LightTheme()

	// デフォルトでvybテーマを使用
	tm.current = VybTheme()

	return tm
}

// テーマを設定
func (tm *ThemeManager) SetTheme(themeType ThemeType) {
	if theme, exists := tm.available[themeType]; exists {
		tm.current = theme
	}
}

// 現在のテーマを取得
func (tm *ThemeManager) GetTheme() Theme {
	return tm.current
}

// 利用可能なテーマ一覧を取得
func (tm *ThemeManager) GetAvailableThemes() []ThemeType {
	themes := make([]ThemeType, 0, len(tm.available))
	for themeType := range tm.available {
		themes = append(themes, themeType)
	}
	return themes
}

// 共通スタイル定義
func (tm *ThemeManager) HeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(tm.current.Primary).
		Bold(true).
		MarginBottom(1)
}

func (tm *ThemeManager) SubheaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(tm.current.Secondary).
		Bold(true)
}

func (tm *ThemeManager) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(tm.current.Success).
		Bold(true)
}

func (tm *ThemeManager) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(tm.current.Error).
		Bold(true)
}

func (tm *ThemeManager) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(tm.current.Warning).
		Bold(true)
}

func (tm *ThemeManager) InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(tm.current.Info)
}

func (tm *ThemeManager) BorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(tm.current.Border).
		Padding(1, 2)
}

func (tm *ThemeManager) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(tm.current.Accent).
		Bold(true)
}
