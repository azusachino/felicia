<script lang="ts">
  import type { Lang } from "@felicia/model"
  import { formatChapters, type PlaybackController } from "./engine"

  let {
    controller,
    journeyTitle,
    lang = "en",
    onClose,
  }: {
    controller: PlaybackController
    journeyTitle: string
    lang?: Lang
    onClose: () => void
  } = $props()

  const i18n = $derived(
    {
      zh: {
        title: "导出旅程视频",
        desc: "利用浏览器原生屏幕录制 API 将旅程回放录制为高清视频，并自动导出视频平台（B站 / YouTube）适用的分集章节时间戳。",
        aspectRatio: "画面比例",
        landscape: "🖥️ 横屏 16:9",
        portrait: "📱 竖屏 9:16",
        chaptersOnly: "仅下载章节 (.txt)",
        startRecord: "开始录制视频",
        recording: "录制中...",
        notSupported: "当前浏览器不支持屏幕/标签页录制 API，您可以下载章节文件。",
        prepare: "正在准备屏幕录制，请在弹窗中选择当前标签页...",
        cancelled: "录制已取消或权限被拒绝。",
        portraitSuffix: "竖屏",
        landscapeSuffix: "横屏",
      },
      ja: {
        title: "旅の動画を書き出し",
        desc: "ブラウザの録画機能を使って旅の再生を高画質動画として記録し、チャプター区切り（.txt）を自動生成します。",
        aspectRatio: "アスペクト比",
        landscape: "🖥️ 横画面 16:9",
        portrait: "📱 縦画面 9:16",
        chaptersOnly: "チャプターのみ保存 (.txt)",
        startRecord: "録画を開始",
        recording: "録画中...",
        notSupported: "このブラウザは画面録画に対応していません。チャプターファイルのみ保存できます。",
        prepare: "画面録画の準備中…共有するタブを選択してください。",
        cancelled: "録画がキャンセルされたか、権限が拒否されました。",
        portraitSuffix: "縦画面",
        landscapeSuffix: "横画面",
      },
      en: {
        title: "Export Trip Video",
        desc: "Record the cinematic journey playback directly in your browser and automatically generate video chapters (.txt) for YouTube / video platforms.",
        aspectRatio: "Aspect Ratio",
        landscape: "🖥️ Landscape 16:9",
        portrait: "📱 Portrait 9:16",
        chaptersOnly: "Download Chapters (.txt)",
        startRecord: "Start Recording",
        recording: "Recording...",
        notSupported: "Screen recording is not supported in this browser. You can still download chapter timestamps.",
        prepare: "Preparing screen recording. Please select this browser tab...",
        cancelled: "Recording was cancelled or permission denied.",
        portraitSuffix: "portrait",
        landscapeSuffix: "landscape",
      },
    }[lang]
  )

  let ratio = $state<"landscape" | "portrait">("landscape")
  let isRecording = $state(false)
  let status = $state<string | null>(null)

  function downloadText(content: string, filename: string) {
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" })
    const url = URL.createObjectURL(blob)
    const a = document.createElement("a")
    a.href = url
    a.download = filename
    a.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  }

  function downloadChapters() {
    const chapters = controller.getChapters()
    const formatted = formatChapters(chapters.length ? chapters : [{ label: journeyTitle, timeSeconds: 0 }])
    downloadText(formatted, `${journeyTitle}-chapters.txt`)
  }

  async function startRecording() {
    if (!navigator.mediaDevices?.getDisplayMedia) {
      status = i18n.notSupported
      return
    }

    try {
      isRecording = true
      status = i18n.prepare

      const stream = await navigator.mediaDevices.getDisplayMedia({
        video: { displaySurface: "browser", frameRate: 30 },
        audio: false,
      })

      const mimeType = ["video/mp4;codecs=avc1", "video/mp4", "video/webm;codecs=vp9", "video/webm"].find((m) =>
        MediaRecorder.isTypeSupported(m)
      )

      const chunks: Blob[] = []
      const recorder = new MediaRecorder(stream, {
        ...(mimeType ? { mimeType } : {}),
        videoBitsPerSecond: 10_000_000,
      })

      recorder.ondataavailable = (e) => {
        if (e.data.size > 0) chunks.push(e.data)
      }

      recorder.onstop = () => {
        isRecording = false
        const ext = recorder.mimeType.includes("mp4") ? "mp4" : "webm"
        const finalBlob = new Blob(chunks, { type: recorder.mimeType || "video/webm" })
        const url = URL.createObjectURL(finalBlob)
        const a = document.createElement("a")
        a.href = url
        a.download = `${journeyTitle}-${ratio === "portrait" ? i18n.portraitSuffix : i18n.landscapeSuffix}.${ext}`
        a.click()
        setTimeout(() => URL.revokeObjectURL(url), 60000)
        downloadChapters()
        onClose()
      }

      // Reset playback to start and play
      controller.seek(0)
      controller.play()
      recorder.start(1000)

      // Listen for user stopping share
      stream.getVideoTracks()[0].addEventListener("ended", () => {
        if (recorder.state === "recording") recorder.stop()
      })
    } catch {
      isRecording = false
      status = i18n.cancelled
    }
  }
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4 backdrop-blur-md">
  <div class="w-full max-w-md rounded-2xl border border-white/20 bg-stone-950 p-6 text-white shadow-2xl">
    <div class="flex items-center justify-between">
      <h2 class="text-base font-bold">{i18n.title}</h2>
      <button
        type="button"
        class="flex size-8 cursor-pointer items-center justify-center rounded-full border-0 bg-transparent text-stone-400 hover:text-white"
        onclick={onClose}
      >
        ✕
      </button>
    </div>

    <p class="mt-2 text-xs leading-relaxed text-stone-400">
      {i18n.desc}
    </p>

    <!-- Ratio Switcher -->
    <div class="mt-4">
      <span class="block text-xs font-medium text-stone-300">{i18n.aspectRatio}</span>
      <div class="mt-2 grid grid-cols-2 gap-2 text-xs">
        <button
          type="button"
          class="flex cursor-pointer items-center justify-center gap-2 rounded-xl border p-2.5 transition {ratio === 'landscape' ? 'border-[#39C5BB] bg-[#39C5BB]/10 text-[#39C5BB]' : 'border-white/10 text-stone-300'}"
          onclick={() => (ratio = "landscape")}
        >
          <span>{i18n.landscape}</span>
        </button>
        <button
          type="button"
          class="flex cursor-pointer items-center justify-center gap-2 rounded-xl border p-2.5 transition {ratio === 'portrait' ? 'border-[#39C5BB] bg-[#39C5BB]/10 text-[#39C5BB]' : 'border-white/10 text-stone-300'}"
          onclick={() => (ratio = "portrait")}
        >
          <span>{i18n.portrait}</span>
        </button>
      </div>
    </div>

    {#if status}
      <p class="mt-3 text-xs text-teal-300">{status}</p>
    {/if}

    <!-- Buttons -->
    <div class="mt-6 flex items-center justify-between gap-3">
      <button
        type="button"
        class="cursor-pointer rounded-xl border border-white/15 bg-white/5 px-3 py-2 text-xs text-stone-300 hover:bg-white/10 hover:text-white"
        onclick={downloadChapters}
      >
        {i18n.chaptersOnly}
      </button>

      <button
        type="button"
        class="cursor-pointer rounded-xl border border-[#39C5BB]/50 bg-[#39C5BB] px-4 py-2 text-xs font-semibold text-zinc-950 shadow-lg transition hover:bg-[#39C5BB]/80 disabled:opacity-50"
        disabled={isRecording}
        onclick={startRecording}
      >
        {isRecording ? i18n.recording : i18n.startRecord}
      </button>
    </div>
  </div>
</div>
