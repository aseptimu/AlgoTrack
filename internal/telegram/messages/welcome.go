package messages

// Commands is the canonical command list shown by /help and the welcome
// flows. Sent with ParseMode HTML, so &lt; / &gt; render as < / >.
const Commands = `<b>Команды</b>
/start — приветствие и сброс к началу
/help — этот список
/add &lt;номер&gt; — отметить решённую задачу или повторение
/goal &lt;число&gt; [easy|medium|hard] — поставить цель
/link &lt;ник_leetcode&gt; — привязать аккаунт LeetCode для авто-трекинга
/list [easy|medium|hard] — список решённых задач
/stats — дашборд прогресса
/next [js] — получить новую рекомендованную задачу
/review — задачи на повторение на сегодня
/mode default|js — переключить источник рекомендаций (NeetCode / 30 Days JS)`

// WelcomeNoGoal is sent on /start when the user has no goals set yet.
// The %s placeholder is filled with Commands by the caller.
const WelcomeNoGoal = `<b>Привет! Это AlgoTrack</b> 🚀

Я помогу тебе:
• Отслеживать решённые задачи на LeetCode
• Повторять их по spaced-repetition расписанию
• Идти к цели (например, 700 задач)

Выбери стартовую цель кнопкой ниже или поставь свою через /goal.

%s`

// Help is returned by /help.
const Help = `<b>AlgoTrack — помощь</b>

` + Commands

const GoalUsage = `Используй команду так:
/goal 300
/goal 100 easy
/goal 300 medium`

const InvalidGoal = `Нужно положительное число.
Пример:
/goal 300
/goal 100 easy
/goal 300 medium`

const InvalidGoalDifficulty = `Допустимые сложности: easy, medium, hard.
Пример:
/goal 100 easy
/goal 300 medium`

const GoalSavedNoProgress = `<b>Цель сохранена</b>.`

const GoalSavedWithProgress = `<b>Цель сохранена</b>.

<b>Прогресс</b>:
%s`

// WelcomeWithProgress is shown to a returning user (already has goals set).
// The trailing %s is filled with Commands by the caller.
func WelcomeWithProgress(progressLines string) string {
	return `<b>С возвращением в AlgoTrack</b>

<b>Цели</b>:
` + progressLines + `

Так держать 🚀

%s`
}
