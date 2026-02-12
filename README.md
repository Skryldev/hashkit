<div dir="rtl">

# 🔐 ماژول hashkit برای Golang

یک ماژول حرفه‌ای و **Production-Ready** برای مدیریت پسوردها در پروژه‌های Golang.
این ماژول به شما امکان می‌دهد پسوردها را با الگوریتم‌های امن **bcrypt** و **Argon2id** هش کنید، اعتبارسنجی کنید و حتی به صورت **auto-rehash** در صورت قدیمی بودن hash، آنها را به‌روزرسانی کنید.

---

## 🌟 ویژگی‌های اصلی

- پشتیبانی از **bcrypt** و **Argon2id**
- مدیریت **cost و memory** برای balancing بین امنیت و performance
- **Auto-rehash** حرفه‌ای در پس‌زمینه (Async)
- struct-based return برای راحتی **DB update**
- امنیت بالا: timing-safe compare و مدیریت hash خراب
- Production-ready: safe concurrency، configurable، reusable

---

## 🛠 نصب

با استفاده از `go get`:

```bash
go get github.com/Skryldev/hashkit
```
---
## 🧩 1️⃣ ساخت Engine (Initialization)
### این کار معمولاً هنگام بالا آمدن برنامه انجام می‌شود (مثلاً داخل main()).

<div dir="ltr">

```go
func main() {

	cfg := hashkit.DefaultConfig()

	engine := hashkit.NewEngine(cfg)
	defer engine.Shutdown()

	// حالا engine آماده استفاده است
}
```

<div dir="rtl">

## 📦 نحوه استفاده
### 🔒 Hash کردن پسورد (Signup)

<div dir="ltr">

```go
hash, err := engine.Hash("MySuperSecret123!")
if err != nil {
	log.Fatal(err)
}
// ذخیره در دیتابیس
fmt.Println(hash)
```

<div dir="rtl">

## 🔎 اعتبارسنجی پسورد (Verify)

<div dir="ltr">

```go
ctx := context.Background()

valid, err := engine.Verify(
	ctx,
	"MySuperSecret123!",
	hashFromDB,
	true, // Enable auto rehash
	func(newHash string) { // If Don't want this callback func set nill

		updateUserHashInDB(userID, newHash) // Undate NewHash to your Database
	},
)

if err != nil {
	log.Fatal(err)
}

if !valid {
	fmt.Println("Invalid credentials")
	return
}

fmt.Println("Login successful")
```

<div dir="rtl">

---
## 🧠 پارامترها دقیقاً یعنی چی؟
| پارامتر | نوع | توضیح کامل |
|---------|-----|------------|
| `ctx` | `context.Context` | این پارامتر برای کنترل چرخه زندگی عملیات است. به‌خصوص در درخواست‌های HTTP کاربرد دارد تا در صورت timeout یا cancel شدن request، عملیات verify متوقف شود و منابع آزاد شوند. |
| `password` | `string` | پسورد خامی که کاربر وارد کرده است و باید با hash ذخیره‌شده مقایسه شود. |
| `storedHash` | `string` | hash ذخیره‌شده در پایگاه‌داده برای کاربر. این hash می‌تواند Argon2id یا bcrypt باشد (در نسخه Hybrid). |
| `autoRehash` | `bool` | اگر این مقدار true باشد و پارامترهای config ماژول (مانند memory، iterations، parallelism) تغییر کرده باشند، پس از verify موفق، hash جدید ساخته می‌شود تا امنیت افزایش یابد. |
| `callback` | `func(newHash string)` | این تابع زمانی فراخوانی می‌شود که hash قدیمی بوده و نیاز به rehash دارد. معمولاً داخل این callback، hash جدید در پایگاه‌داده ذخیره می‌شود. این عملیات در background انجام می‌شود تا login کاربر بلاک نشود. |

## 🏗  استفاده در پروژه واقعی (مثال HTTP Handler)
#### مثال ساده:

<div dir="ltr">

```go
func LoginHandler(engine *hashkit.Engine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		password := r.FormValue("password")
		user := findUserFromDB()

		valid, err := engine.Verify(
			r.Context(),
			password,
			user.PasswordHash,
			true,
			func(newHash string) {
				updateUserHashInDB(user.ID, newHash)
			},
		)

		if err != nil || !valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Write([]byte("Login OK"))
	}
}
```

<div dir="rtl">

## 🔥 تنظیم Performance بر اساس سرور
### مثال برای سرور 8 Core:

<div dir="ltr">

```go
cfg := &hashkit.Config{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 8,
	KeyLength:   32,
	SaltLength:  16,
	Workers:     8,
}
```

<div dir="rtl">
