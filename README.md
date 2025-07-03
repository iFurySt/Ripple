# 🌊 Ripple

**Ripple** 是一个将 Notion 中的笔记内容结构化处理后，自动分发至多个平台（如社交媒体、博客、微信公众号等）的内容自动化分发工具。你写下的是想法，它扩散的是影响力。

---

## ✨ Features

- 📝 **支持 Notion 输入**：通过 API 同步 Notion 笔记
- 🔍 **内容处理**：
  - 标题优化
  - 摘要生成
  - 标签提取
  - 多平台模板渲染
- 📣 **一键多平台分发**：
  - Twitter / X
  - 微信公众号（WeChat Official Account）
  - 小红书（可导出待发布内容）
  - Blog 平台（如 Hugo、Ghost、Notion Blog）
  - 邮件（Mailchimp / Substack）
- 🤖 **AI 助力**（可选）：自动润色、拆分为多条内容、智能摘要

---

## 📦 Architecture

```mermaid
graph TD
  A[Notion 笔记] --> B[内容抓取器]
  B --> C[内容解析与结构化]
  C --> D[多平台模板渲染]
  D --> E[分发模块]
  E --> F1[X]
  E --> F2[公众号]
  E --> F3[Blog]
  E --> F4[邮件/RSS]
