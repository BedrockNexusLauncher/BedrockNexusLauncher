<div dir="rtl">

# ریپوی مارکت‌پلیس Bedrock Nexus

این پوشه قالب محتوای «مارکت‌پلیس» است: یک ریپوی GitHub (مثلاً `marketplace`) که Bedrock Nexus از آن ایندکس و فایل‌های مود/ادان/اسکین را می‌خواند — بدون نیاز به سرور.

## ساختار ریپو

```
marketplace/
├── index.json              ← ایندکس تمام آیتم‌ها (تنها منبع داده لانچر)
├── assets/
│   └── <item-id>/
│       ├── icon.png        ← آیکون آیتم (مربع، حداقل 256x256)
│       └── shot1.png ...   ← اسکرین‌شات‌ها
└── (فایل‌های مود در GitHub Releases آپلود می‌شوند، نه داخل ریپو)
```

- آیکون‌ها و اسکرین‌شات‌ها از طریق jsDelivr سرو می‌شوند:
  `https://cdn.jsdelivr.net/gh/<USER>/<REPO>@main/assets/<id>/icon.png`
- فایل‌های دانلودی (mcaddon/mcpack/mcworld) در **Releases** با تگ مثل `example-addon-v1.0.0` قرار می‌گیرند؛ لینک پایدارشان در `downloadUrl` ثبت می‌شود.

## اسکیمای index.json

```jsonc
{
  "schemaVersion": 1,
  "updatedAt": "2026-08-29T00:00:00Z",
  "items": [
    {
      "id": "example-addon",          // یکتا، حروف/عدد/خط تیره
      "name": "Example Addon",
      "author": "YourName",
      "summary": "توضیح یک‌خطی برای کارت",
      "description": "توضیح کامل (Markdown)",
      "icon": "https://.../icon.png",
      "screenshots": ["https://.../shot1.png"],
      "tags": ["addon", "adventure"],
      "type": "mcaddon",              // mcaddon | mcpack | mcworld | skinpack (قابل گسترش به dll/lip)
      "homepage": "https://github.com/...",
      "versions": [
        {
          "version": "1.0.0",
          "date": "2026-08-29",
          "fileType": "mcaddon",      // پسوند فایل: mcaddon | mcpack | mcworld
          "fileName": "ExampleAddon-1.0.0.mcaddon",
          "downloadUrl": "https://github.com/.../releases/download/...",
          "size": 0,                  // بایت؛ 0 = نامشخص
          "changelog": "توضیح تغییرات نسخه"
        }
      ]
    }
  ]
}
```

## افزودن آیتم جدید

اسکریپت آماده در `scripts/upload-addon.ps1` این کار را خودکار می‌کند (ساخت Release + به‌روزرسانی index.json با `gh` CLI):

```powershell
pwsh scripts/upload-addon.ps1 -Id example-addon -Type mcaddon -Version 1.0.0 -File .\ExampleAddon-1.0.0.mcaddon -Repo "YOUR_USER/marketplace"
```

### نکات

- `type: skinpack` یعنی پک اسکین؛ لانچر آن را در مسیر `skin_packs` نصب می‌کند. فایل آن همچنان `.mcpack` است.
- پس از تغییر `index.json` روی شاخه‌ی `main`، کش jsDelivr حداکثر چند دقیقه‌ای به‌روز می‌شود؛ آدرس ایندکس در خود لانچر قابل تنظیم است.

</div>
