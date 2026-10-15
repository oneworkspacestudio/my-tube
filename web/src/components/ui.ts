import {
  createElement as h,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  Download,
  Clapperboard,
  Menu,
  Search,
  SunMoon,
  Plus,
  Bell,
  House,
  Flame,
  TvMinimalPlay,
  Compass,
  History,
  Clock3,
  ThumbsUp,
  Video,
  CircleUserRound,
  ChartNoAxesCombined,
  ListVideo,
  Settings2,
  Layers3,
  Check,
  X,
  Upload,
  SlidersHorizontal,
  EllipsisVertical,
  Ellipsis,
  RectangleHorizontal,
  Share2,
  ListPlus,
  Eye,
  UsersRound,
  RefreshCw,
  Pencil,
  Waves,
  LogOut,
  FolderOpen,
  ShieldCheck,
  ArrowLeft,
  CloudUpload,
  CirclePlus,
  Play,
  CircleAlert,
  CirclePlay,
} from "lucide-react";
import Hls from "hls.js";
import { useUI } from "../context";
import { Dialog } from "../Dialog.generated";
import { apiURL, sendProgress } from "../useTube";
const icons: any = {
  Download,
  Clapperboard,
  Menu,
  Search,
  SunMoon,
  Plus,
  Bell,
  House,
  Flame,
  TvMinimalPlay,
  Compass,
  History,
  Clock3,
  ThumbsUp,
  Video,
  CircleUserRound,
  ChartNoAxesCombined,
  ListVideo,
  Settings2,
  Layers3,
  Check,
  X,
  Upload,
  SlidersHorizontal,
  EllipsisVertical,
  Ellipsis,
  RectangleHorizontal,
  Share2,
  ListPlus,
  Eye,
  UsersRound,
  RefreshCw,
  Pencil,
  Waves,
  LogOut,
  FolderOpen,
  ShieldCheck,
  ArrowLeft,
  CloudUpload,
  CirclePlus,
  Play,
  CircleAlert,
};
type P = Record<string, any> & { children?: ReactNode };
const cls = (p: P) => [p.region, p.variant, p.tone].filter(Boolean).join(" ");
export function Page(p: P) {
  return h("div", { className: cls(p) || "application" }, p.children);
}
export function Stack(p: P) {
  return h("div", { className: "stack " + cls(p) }, p.children);
}
export function Row(p: P) {
  return h("div", { className: "row " + cls(p) }, p.children);
}
export function Section(p: P) {
  return h(
    p.region === "main" ? "main" : p.region === "sidebar" ? "aside" : "section",
    { className: cls(p) },
    p.children,
  );
}
export function Grid(p: P) {
  return h("div", { className: "grid " + cls(p) }, p.children);
}
export function Card(p: P) {
  const ui = useUI();
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (p.region !== "modal") return;
    const previous = document.activeElement as HTMLElement;
    ref.current
      ?.querySelector<HTMLElement>("button,input,textarea,select")
      ?.focus();
    const key = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        ui.action("close-dialog");
        return;
      }
      if (e.key === "Tab") {
        const list = Array.from(
          ref.current?.querySelectorAll<HTMLElement>(
            "button:not(:disabled),input,textarea,select,a[href]",
          ) || [],
        );
        if (!list.length) return;
        const first = list[0],
          last = list[list.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    document.addEventListener("keydown", key);
    return () => {
      document.removeEventListener("keydown", key);
      previous?.focus();
    };
  }, [p.region]);
  return h(
    "div",
    {
      className: "card " + cls(p),
      ref,
      ...(p.region === "modal"
        ? {
            role: "dialog",
            "aria-modal": true,
            "aria-label": ui.app.dialogTitle,
          }
        : {}),
      ...(p.region === "dropzone"
        ? {
            onDragOver: (e: DragEvent) => e.preventDefault(),
            onDrop: (e: DragEvent) => {
              e.preventDefault();
              ui.field("files", Array.from(e.dataTransfer?.files || []));
            },
          }
        : {}),
    },
    p.children,
  );
}
export function Heading(p: P) {
  return h("h" + (p.level || 2), {}, p.children);
}
export function Text(p: P) {
  return h("span", { className: "text " + cls(p) }, p.children);
}
export function Badge(p: P) {
  return h("span", { className: "badge " + cls(p) }, p.children);
}
export function Icon(p: P) {
  const Component = icons[p.name] || CirclePlay;
  return h(Component, {
    size: p.size || 20,
    "aria-hidden": true,
    strokeWidth: 1.7,
  });
}
export function Button(p: P) {
  const ui = useUI();
  return h(
    "button",
    {
      className: "button " + cls(p),
      type: "button",
      title: p.title,
      disabled: !!p.disabled,
      onClick: () => ui.action(p.intent, p.payload),
    },
    p.icon ? h(Icon, { name: p.icon }) : null,
    p.variant === "channel-avatar" && p.payload
      ? h(Avatar, {
          src: apiURL(`channels/${p.payload}/image/avatar`),
          children: p.children,
        })
      : p.variant === "avatar" && ui.app.avatar
        ? h(Avatar, { src: ui.app.avatar, children: p.children })
        : p.children,
  );
}
function Avatar(p: P) {
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [p.src]);
  if (failed) return p.children;
  return h("img", {
    src: p.src,
    alt: "",
    onError: () => setFailed(true),
    className: "avatar-image",
  });
}
export function Link(p: P) {
  return h(
    "a",
    {
      className: "link " + cls(p),
      href: p.href,
      target: String(p.href).startsWith("https:") ? "_blank" : undefined,
      rel: "noopener noreferrer",
    },
    p.children,
  );
}
export function Divider() {
  return h("hr");
}
export function Spacer() {
  return h("span", { className: "spacer" });
}
export function Alert(p: P) {
  return h("div", { role: "alert", className: "alert " + cls(p) }, p.children);
}
export function Metric(p: P) {
  if (p.tone === "progress")
    return h("progress", {
      value: Math.min(100, Math.max(0, Number(p.value) || 0)),
      max: 100,
    });
  return h("strong", { className: "metric" }, p.value ?? 0);
}
export function Image(p: P) {
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [p.src]);
  if (failed || !p.src)
    return h(
      "div",
      {
        className: "image-fallback " + p.fallback,
        "aria-label": p.alt || "Video thumbnail",
      },
      h(Icon, { name: p.fallback === "banner" ? "Waves" : "Play", size: 40 }),
    );
  return h("img", {
    className: cls(p),
    src: p.src,
    alt: p.alt || "",
    loading: "lazy",
    onError: () => setFailed(true),
  });
}
export function Select(p: P) {
  const ui = useUI();
  return h(
    "label",
    { className: "field select-field" },
    p.title ? h("span", {}, p.title) : null,
    h(
      "select",
      {
        value: p.value ?? "",
        onChange: (e: any) => ui.field(p.field, e.target.value),
        "aria-label": p.title,
      },
      (p.options || []).map((o: any) =>
        h(
          "option",
          {
            key: typeof o === "string" ? o : o.value,
            value: typeof o === "string" ? o : o.value,
          },
          typeof o === "string" ? o : o.label,
        ),
      ),
    ),
  );
}
export function Checkbox(p: P) {
  const ui = useUI();
  const [checked, setChecked] = useState(!!p.checked);
  useEffect(() => setChecked(!!p.checked), [p.checked]);
  return h(
    "label",
    { className: "checkbox" },
    h("input", {
      type: "checkbox",
      checked,
      onChange: (e: any) => {
        setChecked(e.target.checked);
        ui.field(p.field, e.target.checked);
      },
    }),
    p.children,
  );
}
export function Input(p: P) {
  const ui = useUI();
  if (p.kind === "channelAvatar")
    return h(
      "span",
      { className: "badge large-avatar" },
      h(Avatar, { src: p.src, children: p.text }),
    );
  if (p.kind === "dialog")
    return h(Dialog, { data: { app: ui.app }, onAction: ui.action });
  if (p.kind === "player") return h(Player, { video: p.video });
  if (p.kind === "select") return h(Select, p);
  const file = p.kind === "files" || p.kind === "file";
  const attrs: any = {
    value: file ? undefined : (p.value ?? ""),
    placeholder: p.placeholder,
    "aria-label": p.title || p.placeholder,
    onChange: (e: any) =>
      ui.field(
        p.field,
        file
          ? p.multiple
            ? Array.from(e.target.files || [])
            : e.target.files?.[0]
          : e.target.value,
      ),
  };
  if (file) {
    attrs.type = "file";
    attrs.accept = p.accept;
    attrs.multiple = !!p.multiple;
  }
  if (p.kind === "password") attrs.type = "password";
  if (p.kind === "number") attrs.type = "number";
  attrs.onKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Enter" && p.field === "search") ui.action("search");
  };
  if (p.kind === "textarea") attrs.rows = p.rows || 3;
  return h(
    "label",
    { className: "field " + (p.kind || "") },
    p.title && p.field !== "search" ? h("span", {}, p.title) : null,
    h(p.kind === "textarea" ? "textarea" : "input", attrs),
  );
}
function Player({ video: v }: any) {
  const ui = useUI();
  const ref = useRef<HTMLVideoElement>(null);
  const hls = useRef<Hls | null>(null);
  const [levels, setLevels] = useState<any[]>([]);
  const [quality, setQuality] = useState("-1");
  const [speed, setSpeed] = useState("1");
  const [error, setError] = useState("");
  const last = useRef(0);
  const uid = ui.app.signedIn;
  const csrf = ui.app.csrf;
  useEffect(() => {
    const el = ref.current;
    if (!el || !v?.id) return;
    setError("");
    setLevels([]);
    setQuality("-1");
    last.current = 0;
    let engine: Hls | null = null;
    if (v.hls && Hls.isSupported()) {
      engine = new Hls({ maxBufferLength: 30 });
      hls.current = engine;
      engine.loadSource(apiURL(`videos/${v.id}/media/master.m3u8`));
      engine.attachMedia(el);
      engine.on(Hls.Events.MANIFEST_PARSED, () => setLevels(engine!.levels));
      engine.on(Hls.Events.ERROR, (_e, data) => {
        if (data.fatal)
          setError(
            "Streaming interrupted. Retry playback or choose Original if available.",
          );
      });
    } else if (v.hls && el.canPlayType("application/vnd.apple.mpegurl")) {
      el.src = apiURL(`videos/${v.id}/media/master.m3u8`);
    } else if (v.direct) {
      el.src = apiURL(`videos/${v.id}/media/original`);
    } else {
      setError(
        v.error ||
          "This video is waiting for a compatible streaming rendition.",
      );
    }
    const loaded = () => {
      if (v.position > 0 && v.position < el.duration - 3)
        el.currentTime = v.position;
      el.playbackRate = Number(speed);
    };
    el.addEventListener("loadedmetadata", loaded);
    return () => {
      el.pause();
      engine?.destroy();
      hls.current = null;
      el.removeEventListener("loadedmetadata", loaded);
      el.removeAttribute("src");
      el.load();
    };
  }, [v?.id, v?.hls, v?.direct]);
  useEffect(() => {
    const el = ref.current;
    if (!el || !v?.id || !uid) return;
    const checkpoint = () => {
      if (el.paused || !Number.isFinite(el.currentTime)) return;
      const position = el.currentTime;
      const seconds = Math.min(10, Math.max(0, position - last.current));
      last.current = position;
      void sendProgress(v.id, { position, seconds }, csrf);
    };
    const t = window.setInterval(checkpoint, 5000);
    return () => window.clearInterval(t);
  }, [v?.id, uid, csrf]);
  useEffect(() => {
    const seek = (e: Event) => {
      if (ref.current) ref.current.currentTime = (e as CustomEvent).detail;
    };
    window.addEventListener("mytube-seek", seek);
    return () => window.removeEventListener("mytube-seek", seek);
  }, []);
  return h(
    "div",
    { className: "video-player" },
    h(
      "video",
      {
        ref,
        controls: true,
        playsInline: true,
        preload: "metadata",
        poster: v?.poster,
        onError: () =>
          setError(
            "Playback could not start. Check the processing status or try another quality.",
          ),
        onEnded: () => ui.action("ended"),
        "aria-label": v?.title || "Video player",
      },
      h("track", {
        src: apiURL(`videos/${v?.id}/media/captions.vtt`),
        kind: "subtitles",
        srcLang: "en",
        label: "Uploaded captions",
      }),
    ),
    error
      ? h(
          "div",
          { className: "playback-error", role: "status" },
          h(Icon, { name: "CircleAlert" }),
          error,
        )
      : null,
    h(
      "div",
      { className: "playback-options" },
      h(
        "label",
        {},
        "Quality ",
        h(
          "select",
          {
            "aria-label": "Playback quality",
            value: quality,
            onChange: (e: any) => {
              setQuality(e.target.value);
              const n = Number(e.target.value);
              if (n === -2 && v.direct) {
                hls.current?.destroy();
                hls.current = null;
                const el = ref.current!;
                const position = el.currentTime;
                el.src = apiURL(`videos/${v.id}/media/original`);
                el.addEventListener(
                  "loadedmetadata",
                  () => {
                    el.currentTime = position;
                    void el.play();
                  },
                  { once: true },
                );
              } else if (hls.current) {
                hls.current.currentLevel = n;
              }
            },
          },
          h("option", { value: "-1" }, v.hls ? "Auto" : "Original"),
          ...levels.map((l, i) =>
            h("option", { value: String(i), key: i }, `${l.height}p`),
          ),
        ),
      ),
      h(
        "label",
        {},
        "Speed ",
        h(
          "select",
          {
            "aria-label": "Playback speed",
            value: speed,
            onChange: (e: any) => {
              setSpeed(e.target.value);
              if (ref.current)
                ref.current.playbackRate = Number(e.target.value);
            },
          },
          ["0.5", "0.75", "1", "1.25", "1.5", "2"].map((n) =>
            h("option", { key: n, value: n }, n + "×"),
          ),
        ),
      ),
    ),
  );
}
