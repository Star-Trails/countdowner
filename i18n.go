package main

import "slices"

// Keep the user's requested language order, independent of the current locale.
var languageCodes = []string{"en", "zh-Hant", "zh-Hans", "ja", "ko"}
var languageNames = []string{"English", "正體中文", "简体中文", "日本語", "한국어"}

// Columns are Traditional Chinese, Simplified Chinese, Japanese, and Korean.
// The dictionary key itself is the English translation.
var translations = map[string][4]string{
	"About":                           {"關於", "关于", "このアプリについて", "정보"},
	"Settings":                        {"設定", "设置", "設定", "설정"},
	"Quit":                            {"結束", "退出", "終了", "종료"},
	"Your deadlines, on your desktop": {"重要時刻，就在桌面", "重要时刻，就在桌面", "大切な予定をデスクトップに", "중요한 일정을 바탕 화면에"},
	"Countdowns":                      {"倒數計時", "倒计时", "カウントダウン", "카운트다운"},
	"No countdowns yet":               {"尚未新增倒數計時", "尚未添加倒计时", "カウントダウンはまだありません", "아직 카운트다운이 없습니다"},
	"Open settings from the tray to add a deadline": {"從系統匣開啟設定，新增重要時刻", "从托盘打开设置，添加重要时刻", "トレイから設定を開き、予定を追加", "트레이에서 설정을 열어 일정을 추가하세요"},
	"New countdown":        {"新增倒數計時", "新增倒计时", "新しいカウントダウン", "새 카운트다운"},
	"Edit countdown":       {"編輯倒數計時", "编辑倒计时", "カウントダウンを編集", "카운트다운 편집"},
	"Title":                {"標題", "标题", "タイトル", "제목"},
	"Target date and time": {"目標日期與時間", "目标日期与时间", "目標日時", "목표 날짜와 시간"},
	"Date":                 {"日期", "日期", "日付", "날짜"},
	"Today":                {"今天", "今天", "今日", "오늘"},
	"Tomorrow":             {"明天", "明天", "明日", "내일"},
	"Date only (00:00)":    {"僅日期（00:00）", "仅日期（00:00）", "日付のみ（00:00）", "날짜만 (00:00)"},
	"Hour":                 {"小時", "小时", "時", "시"},
	"Minute":               {"分鐘", "分钟", "分", "분"},
	"Choose hour":          {"選擇小時", "选择小时", "時を選択", "시 선택"},
	"Choose minute":        {"選擇分鐘", "选择分钟", "分を選択", "분 선택"},
	"Hour input":           {"輸入小時", "输入小时", "時を入力", "시 입력"},
	"Minute input":         {"輸入分鐘", "输入分钟", "分を入力", "분 입력"},
	"Date only means 00:00. Times use your local timezone.": {"僅日期代表當日 00:00；時間使用本地時區。", "仅日期代表当天 00:00；时间使用本地时区。", "日付のみは当日の 00:00。時刻はローカル時間です。", "날짜만 선택하면 당일 00:00입니다. 현지 시간대를 사용합니다."},
	"Add countdown":                    {"新增", "添加", "追加", "추가"},
	"Save changes":                     {"儲存變更", "保存更改", "変更を保存", "변경 저장"},
	"Cancel":                           {"取消", "取消", "キャンセル", "취소"},
	"Edit":                             {"編輯", "编辑", "編集", "편집"},
	"Reorder":                          {"重新排序", "重新排序", "並べ替え", "순서 변경"},
	"Countdown indicator":              {"倒數計時圖示", "倒计时图示", "カウントダウン表示", "카운트다운 표시"},
	"Delete":                           {"刪除", "删除", "削除", "삭제"},
	"Confirm delete":                   {"確認刪除", "确认删除", "削除を確認", "삭제 확인"},
	"Move up":                          {"上移", "上移", "上へ", "위로"},
	"Move down":                        {"下移", "下移", "下へ", "아래로"},
	"Manual order":                     {"手動排序", "手动排序", "手動順", "수동 순서"},
	"Deadline, earliest first":         {"依截止時間，由近至遠", "按截止时间，由近至远", "締切の早い順", "마감이 빠른 순서"},
	"Appearance":                       {"外觀", "外观", "外観", "모양"},
	"Theme":                            {"主題", "主题", "テーマ", "테마"},
	"Follow system":                    {"跟隨系統", "跟随系统", "システムに合わせる", "시스템 설정 사용"},
	"Light":                            {"淺色", "浅色", "ライト", "밝게"},
	"Dark":                             {"深色", "深色", "ダーク", "어둡게"},
	"Language":                         {"語言", "语言", "言語", "언어"},
	"Start at login":                   {"登入時自動啟動", "开机自启动", "ログイン時に起動", "로그인 시 시작"},
	"Reset desktop position":           {"重設桌面位置", "重置桌面位置", "デスクトップの位置をリセット", "바탕 화면 위치 초기화"},
	"Saved":                            {"已儲存", "已保存", "保存しました", "저장됨"},
	"Expired":                          {"已到期", "已到期", "期限切れ", "기한 만료"},
	"Enter a title":                    {"請輸入標題", "请输入标题", "タイトルを入力してください", "제목을 입력하세요"},
	"Enter a valid local date or time": {"請輸入有效的本地日期或時間", "请输入有效的本地日期或时间", "有効なローカル日時を入力してください", "유효한 현지 날짜 또는 시간을 입력하세요"},
	"Configuration is read-only; the original file is preserved": {"設定為唯讀；原始檔案已保留", "配置只读；原文件已保留", "設定は読み取り専用です。元のファイルは保持されています", "설정은 읽기 전용입니다. 원본 파일은 보존됩니다"},
	"Desktop connected":            {"已連接桌面", "已连接桌面", "デスクトップに接続済み", "바탕 화면에 연결됨"},
	"Waiting for Explorer desktop": {"正在等待 Explorer 桌面", "正在等待 Explorer 桌面", "Explorer デスクトップを待機中", "Explorer 바탕 화면 대기 중"},
	"Data folder":                  {"資料夾", "数据目录", "データフォルダー", "데이터 폴더"},
	"Automatically saved. Closing this window keeps the widget running.": {"變更自動儲存。關閉此視窗不會結束桌面元件。", "更改自动保存。关闭此窗口不会退出桌面组件。", "変更は自動保存されます。このウィンドウを閉じても動作し続けます。", "변경 사항은 자동 저장됩니다. 이 창을 닫아도 위젯은 계속 실행됩니다."},
}

func (a *application) t(key string) string {
	index := slices.Index(languageCodes, a.config.Language)
	if index > 0 {
		if values, ok := translations[key]; ok {
			return values[index-1]
		}
	}
	return key
}

func (a *application) fontFamily() string {
	switch a.config.Language {
	case "zh-Hant":
		return "Microsoft JhengHei UI, system-ui"
	case "zh-Hans":
		return "Microsoft YaHei UI, system-ui"
	case "ja":
		return "Yu Gothic UI, system-ui"
	case "ko":
		return "Malgun Gothic, system-ui"
	default:
		return "system-ui"
	}
}
