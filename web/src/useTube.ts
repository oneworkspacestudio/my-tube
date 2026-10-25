import { useEffect, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
export const base =
  document.querySelector<HTMLMetaElement>('meta[name="mytube-base"]')
    ?.content || "/";
export const apiURL = (p: string) => base + "api/" + p;
async function request(
  p: string,
  method = "GET",
  body?: any,
  csrf = "",
  headers: Record<string, string> = {},
) {
  const options: RequestInit = {
    method,
    credentials: "same-origin",
    headers: {
      ...(method === "GET" ? {} : { "X-CSRF-Token": csrf }),
      ...headers,
    },
  };
  if (body !== undefined) {
    if (body instanceof Blob) {
      options.body = body;
    } else {
      options.body = JSON.stringify(body);
      (options.headers as any)["Content-Type"] = "application/json";
    }
  }
  const r = await fetch(apiURL(p), options);
  let result: any;
  try {
    result = await r.json();
  } catch {
    throw Error("The server returned an unreadable response");
  }
  if (!r.ok) throw Error(result.error || `Request failed (${r.status})`);
  return result;
}
export async function sendProgress(id: string, data: any, csrf: string) {
  try {
    await request(`videos/${id}/progress`, "POST", data, csrf);
  } catch {
    /* Next checkpoint retries. */
  }
}
const initials = (s = "") =>
  s
    .split(/\s+/)
    .slice(0, 2)
    .map((x) => x[0])
    .join("")
    .toUpperCase() || "MT";
const duration = (n = 0) =>
  n >= 3600
    ? `${Math.floor(n / 3600)}:${String(Math.floor(n / 60) % 60).padStart(2, "0")}:${String(Math.floor(n) % 60).padStart(2, "0")}`
    : `${Math.floor(n / 60)}:${String(Math.floor(n) % 60).padStart(2, "0")}`;
const relative = (ts: number) => {
  const days = Math.max(0, Math.floor((Date.now() - ts) / 86400000));
  return days === 0
    ? "Today"
    : days === 1
      ? "Yesterday"
      : days < 30
        ? `${days} days ago`
        : new Date(ts).toLocaleDateString(undefined, {
            month: "short",
            year: "numeric",
          });
};
function decorate(v: any) {
  return {
    ...v,
    poster: apiURL(`videos/${v.id}/media/poster.jpg?state=${v.status}`),
    initials: initials(v.channel),
    durationLabel: duration(v.duration),
    stats: `${Number(v.views || 0).toLocaleString()} views · ${relative(v.created)}`,
    progress: v.duration ? (100 * v.position) / v.duration : 0,
    statusLabel: `${v.status} · ${v.visibility} · ${(v.size / 1048576).toFixed(1)} MiB`,
  };
}
const visibilityOptions = [
  { value: "private", label: "Private — only you and invited viewers" },
  { value: "unlisted", label: "Unlisted — direct link only" },
  { value: "public", label: "Public — discoverable" },
];
export function useTube() {
  const client = useQueryClient();
  const [route, setRoute] = useState(location.hash.slice(1) || "home");
  const [search, setSearch] = useState("");
  const [applied, setApplied] = useState("");
  const [category, setCategory] = useState("All");
  const [sort, setSort] = useState("recent");
  const [days, setDays] = useState("");
  const [dur, setDur] = useState("");
  const [showFilters, setShowFilters] = useState(false);
  const [menu, setMenu] = useState(true);
  const [cinema, setCinema] = useState(false);
  const [autoplay, setAutoplay] = useState(false);
  const [error, setError] = useState("");
  const [toast, setToast] = useState("");
  const [dialog, setDialog] = useState<any>(null);
  const [form, setForm] = useState<any>({});
  const [comment, setComment] = useState("");
  const [reply, setReply] = useState("");
  const [uploadVisibility, setUploadVisibility] = useState("private");
  const [tasks, setTasks] = useState<any[]>([]);
  const [adminConfig, setAdminConfig] = useState("");
  const [channelTab, setChannelTab] = useState("videos");
  const [importBusy, setImportBusy] = useState(false);
  const files = useRef(new Map<string, File>());
  const active = useRef(new Set<string>());
  const go = (to: string) => {
    location.hash = to;
    setRoute(to);
    setError("");
    window.scrollTo(0, 0);
  };
  useEffect(() => {
    const f = () => {
      setRoute(location.hash.slice(1) || "home");
      setError("");
      window.scrollTo(0, 0);
    };
    window.addEventListener("hashchange", f);
    return () => window.removeEventListener("hashchange", f);
  }, []);
  useEffect(() => {
    document.documentElement.dataset.theme =
      localStorage.getItem("mytube-theme") || "light";
  }, []);
  const [screen, id] = route.split("/");
  const meQ = useQuery({
    queryKey: ["me"],
    queryFn: () => request("me"),
    refetchInterval: 5000,
  });
  const me = meQ.data || {};
  const signedIn = !!me.user?.id;
  const csrf = me.csrf || "";
  const uid = me.user?.id || "";
  const query = new URLSearchParams({
    view: screen === "explore" || screen === "playlist" ? "home" : screen,
    q: applied,
    category,
    sort,
    days,
    duration: dur,
  });
  if (screen === "playlist") query.set("playlist", id || "");
  if (screen === "channel") query.set("channel", id || "");
  const videosQ = useQuery<any[]>({
    queryKey: ["videos", query.toString(), uid],
    queryFn: () => request("videos?" + query),
    refetchInterval: screen === "studio" || screen === "channel" ? 5000 : false,
  });
  const watchQ = useQuery({
    queryKey: ["video", id, uid],
    queryFn: () => request("videos/" + id),
    enabled: screen === "watch" && !!id,
    refetchInterval: screen === "watch" ? 10000 : false,
  });
  const recQ = useQuery<any[]>({
    queryKey: ["recommendations", uid],
    queryFn: () => request("videos?recommend=1"),
    enabled: screen === "watch",
  });
  const commentsQ = useQuery<any[]>({
    queryKey: ["comments", id, uid],
    queryFn: () => request(`videos/${id}/comments`),
    enabled: screen === "watch" && !!id,
  });
  const playlistsQ = useQuery<any[]>({
    queryKey: ["playlists", uid],
    queryFn: () => request("playlists"),
    enabled: signedIn,
  });
  const channelQ = useQuery({
    queryKey: ["channel", id || me.channel_id, uid],
    queryFn: () => request("channels/" + (id || me.channel_id)),
    enabled: screen === "channel" && !!(id || me.channel_id),
    refetchInterval: screen === "channel" ? 3000 : false,
  });
  const channelListsQ = useQuery<any[]>({
    queryKey: ["channel-playlists", channelQ.data?.owner, uid],
    queryFn: () =>
      request("playlists?owner=" + encodeURIComponent(channelQ.data.owner)),
    enabled: screen === "channel" && !!channelQ.data?.owner,
  });
  const analyticsQ = useQuery({
    queryKey: ["studio", uid],
    queryFn: () => request("studio"),
    enabled: screen === "studio" && signedIn,
    refetchInterval: 5000,
  });
  const notificationsQ = useQuery<any[]>({
    queryKey: ["notifications", uid],
    queryFn: () => request("notifications"),
    enabled: screen === "notifications" && signedIn,
  });
  const adminQ = useQuery({
    queryKey: ["admin", uid],
    queryFn: () => request("admin"),
    enabled: screen === "admin" && !!me.user?.admin,
  });
  const uploadsQ = useQuery<any[]>({
    queryKey: ["uploads", uid],
    queryFn: () => request("uploads"),
    enabled: screen === "upload" && signedIn,
  });
  const importsQ = useQuery<any[]>({
    queryKey: ["youtube", uid],
    queryFn: () => request("youtube"),
    enabled: screen === "imports" && signedIn,
    refetchInterval: screen === "imports" ? 1500 : false,
  });
  const channelsQ = useQuery<any[]>({
    queryKey: ["channels", uid],
    queryFn: () => request("channels"),
    enabled: signedIn,
    refetchInterval: screen === "imports" ? 3000 : false,
  });
  useEffect(() => {
    if (adminQ.data && !adminConfig)
      setAdminConfig(JSON.stringify(adminQ.data.config, null, 2));
  }, [adminQ.data]);
  useEffect(() => {
    if (uploadsQ.data)
      setTasks((old) => {
        const out = [...old];
        for (const u of uploadsQ.data!) {
          if (!out.some((x) => x.id === u.id))
            out.push({
              ...u,
              percent: (100 * u.offset) / u.size,
              label: "Paused — select the same file",
            });
        }
        return out;
      });
  }, [uploadsQ.data]);
  const refresh = () => client.invalidateQueries();
  const write = (p: string, m = "POST", b?: any, headers?: any) =>
    request(p, m, b, csrf, headers);
  const show = (
    kind: string,
    title: string,
    fields: any[] = [],
    actions: any[] = [],
    message = "",
  ) => {
    setForm(Object.fromEntries(fields.map((f) => [f.key, f.value ?? ""])));
    setDialog({ kind, title, fields, actions, message });
  };
  const f = (
    key: string,
    label: string,
    value: any = "",
    kind = "text",
    options?: any,
  ) => ({ key, label, value, kind, options });
  const btn = (
    intent: string,
    label: string,
    variant = "primary",
    payload?: any,
  ) => ({ intent, label, variant, payload });
  const need = () => {
    if (!signedIn) throw Error("Sign in with MyAccounts to continue");
  };
  function editChannel() {
    const c = channelQ.data || {};
    show(
      "channel",
      me.channel_id ? "Customize your channel" : "Create your channel",
      [
        f("name", "Channel name", c.name || me.user?.name || ""),
        f("handle", "Handle (3–40 letters, numbers, _ or -)", c.handle || ""),
        f("description", "About your channel", c.description || "", "textarea"),
        f(
          "visibility",
          "Channel visibility",
          c.visibility || "public",
          "select",
          ["public", "private"],
        ),
        f("avatar", "Avatar (JPEG / PNG)", null, "file"),
        f("banner", "Banner (JPEG / PNG)", null, "file"),
      ],
      [btn("save-channel", "Save channel")],
    );
  }
  function editVideo(v: any) {
    show(
      "edit-video",
      "Video details",
      [
        f("title", "Title", v.title),
        f("description", "Description", v.description, "textarea"),
        f("tags", "Tags, separated by commas", v.tags),
        f("category", "Category", v.category, "select", [
          "General",
          "Music",
          "Learning",
          "Technology",
          "Travel",
          "Gaming",
          "Film",
          "Nature",
        ]),
        f(
          "visibility",
          "Visibility",
          v.visibility,
          "select",
          visibilityOptions.filter(
            (o) => me.public_uploads || o.value !== "public",
          ),
        ),
        f(
          "chapters",
          'Chapters JSON: [{"time":0,"title":"Intro"}]',
          v.chapters === "null" ? "[]" : v.chapters,
          "textarea",
        ),
        f("thumbnail", "Custom thumbnail (JPEG / PNG)", null, "file"),
        f("subtitles", "Captions (.vtt)", null, "file"),
        f("replacement", "Replacement video file (optional)", null, "file"),
      ],
      [
        btn("save-video", "Save details", "primary", v.id),
        btn(
          "replace-video",
          "Replace video with selected file",
          "outline",
          v.id,
        ),
        btn("delete-video", "Delete video", "danger", v.id),
      ],
      "Private video access and original downloads are always checked on the server.",
    );
  }
  async function upload(file: File, existing?: any) {
    need();
    let u = existing;
    try {
      if (!u)
        u = await write("uploads", "POST", {
          name: file.name,
          size: file.size,
          title: file.name.replace(/\.[^.]+$/, ""),
          visibility: uploadVisibility,
        });
      if (active.current.has(u.id)) return;
      active.current.add(u.id);
      files.current.set(u.id, file);
      const update = (p: any) =>
        setTasks((old) => {
          const item = { ...u, ...p };
          return old.some((x) => x.id === u.id)
            ? old.map((x) => (x.id === u.id ? { ...x, ...item } : x))
            : [...old, item];
        });
      update({
        percent: (100 * u.offset) / u.size,
        label: "Uploading",
        error: "",
      });
      let failures = 0;
      while (u.offset < file.size) {
        try {
          const chunk = file.slice(
            u.offset,
            Math.min(file.size, u.offset + 8 * 1024 * 1024),
          );
          u = await write(`uploads/${u.id}`, "PATCH", chunk, {
            "Upload-Offset": String(u.offset),
          });
          update({ percent: (100 * u.offset) / u.size, label: "Uploading" });
          failures = 0;
        } catch (e) {
          if (++failures > 3) throw e;
          await new Promise((resolve) =>
            setTimeout(resolve, 500 * 2 ** failures),
          );
          u = await request(`uploads/${u.id}`);
        }
      }
      const v = await write(`uploads/${u.id}/complete`, "POST", {});
      update({ percent: 100, label: "Uploaded — processing", video_id: v.id });
      setToast("Video uploaded. Check Creator Studio for processing status.");
      await refresh();
      return v.id;
    } catch (e: any) {
      setError(e.message);
      if (u)
        setTasks((old) =>
          old.map((x) =>
            x.id === u.id
              ? { ...x, label: "Paused — retry available", error: e.message }
              : x,
          ),
        );
    } finally {
      if (u) active.current.delete(u.id);
    }
  }
  async function action(name: string, payload?: any) {
    try {
      setError("");
      switch (name) {
        case "imports":
          need();
          go("imports");
          return;
        case "youtube-add":
          need();
          show(
            "youtube",
            "Import a YouTube channel or playlist",
            [
              f("youtubeURL", "Channel URL, @handle or playlist URL"),
              f(
                "youtubeLatest",
                "Number of recent videos (1–50)",
                10,
                "number",
              ),
              f(
                "youtubeOrder",
                "Playlist window (channels always use newest uploads)",
                "last",
                "select",
                [
                  {
                    value: "last",
                    label: "End of playlist — typically newest additions",
                  },
                  {
                    value: "first",
                    label: "Start of playlist — for newest-first playlists",
                  },
                ],
              ),
              f("youtubeAuto", "Check for new videos", "manual", "select", [
                "manual",
                "daily",
              ]),
            ],
            [btn("youtube-save", "Add and download", "primary")],
            "A channel page is created immediately. Videos download at up to 480p and remain private, along with the matching playlist. Daily checks run while My Tube is running.",
          );
          return;
        case "youtube-save":
          if (importBusy) return;
          setImportBusy(true);
          try {
            await write("youtube", "POST", {
              url: form.youtubeURL,
              latest: Number(form.youtubeLatest),
              playlist_order: form.youtubeOrder,
              auto_hours: form.youtubeAuto === "daily" ? 24 : 0,
            });
            setDialog(null);
            go("imports");
            setToast(
              "Source added. Videos download in the background; you can leave this page.",
            );
            await refresh();
          } finally {
            setImportBusy(false);
          }
          return;
        case "youtube-sync":
        case "youtube-cancel":
        case "youtube-remove":
          if (importBusy) return;
          setImportBusy(true);
          try {
            await write(
              `youtube/${payload}${name === "youtube-remove" ? "" : name === "youtube-sync" ? "/sync" : "/cancel"}`,
              name === "youtube-remove" ? "DELETE" : "POST",
              {},
            );
            setToast(
              name === "youtube-remove"
                ? "Source removed. Downloaded videos and playlist are kept."
                : name === "youtube-cancel"
                  ? "Cancelling import. Saved videos are kept."
                  : "Checking for new videos in the background.",
            );
            await refresh();
          } finally {
            setImportBusy(false);
          }
          return;
        case "youtube-settings": {
          const source = importsQ.data?.find((s: any) => s.id === payload);
          if (!source) return;
          show(
            "youtube-settings",
            "Import settings",
            [
              f(
                "youtubeLatest",
                "Number of recent videos (1–50)",
                source.latest,
                "number",
              ),
              f(
                "youtubeOrder",
                "Playlist window (channels always use newest uploads)",
                source.playlist_order,
                "select",
                [
                  {
                    value: "last",
                    label: "End of playlist — typically newest additions",
                  },
                  {
                    value: "first",
                    label: "Start of playlist — for newest-first playlists",
                  },
                ],
              ),
              f(
                "youtubeAuto",
                "Check for new videos",
                source.auto_hours ? "daily" : "manual",
                "select",
                ["manual", "daily"],
              ),
            ],
            [btn("youtube-settings-save", "Save settings", "primary", payload)],
            "Changing this does not remove existing videos. Choose Check for new videos to run immediately.",
          );
          return;
        }
        case "youtube-settings-save":
          if (importBusy) return;
          setImportBusy(true);
          try {
            await write(`youtube/${payload}`, "PUT", {
              latest: Number(form.youtubeLatest),
              playlist_order: form.youtubeOrder,
              auto_hours: form.youtubeAuto === "daily" ? 24 : 0,
            });
            setDialog(null);
            await refresh();
          } finally {
            setImportBusy(false);
          }
          return;
        case "menu":
          setMenu((x) => !x);
          return;
        case "theme": {
          const dark = document.documentElement.dataset.theme !== "dark";
          document.documentElement.dataset.theme = dark ? "dark" : "light";
          localStorage.setItem("mytube-theme", dark ? "dark" : "light");
          return;
        }
        case "home":
          setApplied("");
          setSearch("");
          setCategory("All");
          go("home");
          return;
        case "navigate":
          go(payload);
          return;
        case "watch":
          go("watch/" + payload);
          return;
        case "visit-channel":
          setChannelTab("videos");
          go("channel/" + payload);
          return;
        case "search":
          setApplied(search);
          go("home");
          return;
        case "category":
          setCategory(payload);
          return;
        case "filters":
          setShowFilters((x) => !x);
          return;
        case "cinema":
          setCinema((x) => !x);
          return;
        case "dismiss":
          setToast("");
          return;
        case "close-dialog":
          setDialog(null);
          return;
        case "settings":
          go("settings");
          return;
        case "notifications":
          need();
          go("notifications");
          return;
        case "studio":
          need();
          if (!me.channel_id) {
            editChannel();
            return;
          }
          go("studio");
          return;
        case "channel":
          need();
          if (!me.channel_id) {
            editChannel();
            return;
          }
          go("channel/" + me.channel_id);
          return;
        case "edit-channel":
          need();
          editChannel();
          return;
        case "save-channel": {
          const result = await write("channel", "POST", {
            name: form.name,
            handle: form.handle,
            description: form.description,
            visibility: form.visibility,
          });
          for (const kind of ["avatar", "banner"])
            if (form[kind] instanceof File)
              await write("channel/image/" + kind, "POST", form[kind]);
          setDialog(null);
          await refresh();
          go("channel/" + result.id);
          return;
        }
        case "channel-videos":
          setChannelTab("videos");
          return;
        case "channel-playlists":
          setChannelTab("playlists");
          return;
        case "channel-about":
          setChannelTab("about");
          return;
        case "upload":
          need();
          if (!me.channel_id) {
            editChannel();
            return;
          }
          go("upload");
          return;
        case "resume-upload": {
          const old = tasks.find((u) => u.id === payload);
          const cached = files.current.get(payload);
          if (cached) {
            void upload(cached, old);
            return;
          }
          show(
            "resume-upload",
            "Resume upload",
            [f("resumeFile", "Select the original video file", null, "file")],
            [btn("resume-confirm", "Resume upload", "primary", payload)],
            `${old?.name} · ${old?.size} bytes. Previously acknowledged chunks are kept.`,
          );
          return;
        }
        case "resume-confirm": {
          const old = tasks.find((u) => u.id === payload);
          if (
            !(form.resumeFile instanceof File) ||
            form.resumeFile.name !== old.name ||
            form.resumeFile.size !== old.size
          )
            throw Error("Select the same file with the same filename and size");
          setDialog(null);
          void upload(form.resumeFile, await request("uploads/" + payload));
          return;
        }
        case "cancel-upload":
          if (active.current.has(payload))
            throw Error(
              "Wait for the current chunk to finish before cancelling",
            );
          await write("uploads/" + payload, "DELETE");
          setTasks((old) => old.filter((x) => x.id !== payload));
          return;
        case "edit-video": {
          const v = await request("videos/" + payload);
          editVideo(v);
          return;
        }
        case "save-video": {
          let chapters;
          try {
            chapters = JSON.parse(form.chapters || "[]");
            if (!Array.isArray(chapters)) throw Error();
          } catch {
            throw Error("Chapters must be a JSON array");
          }
          await write("videos/" + payload, "PUT", {
            title: form.title,
            description: form.description,
            tags: form.tags,
            category: form.category,
            visibility: form.visibility,
            chapters,
          });
          if (form.thumbnail instanceof File)
            await write(`videos/${payload}/thumbnail`, "POST", form.thumbnail);
          if (form.subtitles instanceof File)
            await write(`videos/${payload}/subtitles`, "POST", form.subtitles);
          setDialog(null);
          setToast("Video details saved");
          await refresh();
          return;
        }
        case "replace-video": {
          if (!(form.replacement instanceof File))
            throw Error("Choose a replacement video file first");
          const old = await request("videos/" + payload);
          const file = form.replacement;
          const u = await write("uploads", "POST", {
            name: file.name,
            size: file.size,
            title: old.title,
            visibility: "private",
          });
          const replacement = await upload(file, u);
          if (!replacement)
            throw Error(
              "Replacement upload did not finish. The existing video is unchanged.",
            );
          const result = await write("videos/" + payload + "/replace", "POST", {
            replacement_id: replacement,
          });
          setDialog(null);
          await refresh();
          go("watch/" + result.id);
          setToast(
            "Video replaced. The replacement has a new URL; the old original remains recoverable.",
          );
          return;
        }
        case "delete-video":
          show(
            "confirm-delete",
            "Delete this video?",
            [],
            [btn("confirm-delete", "Delete video", "danger", payload)],
            "Playback access stops immediately. Originals remain recoverable until an administrator explicitly purges deleted media.",
          );
          return;
        case "confirm-delete":
          await write("videos/" + payload, "DELETE");
          setDialog(null);
          await refresh();
          go("studio");
          return;
        case "retry":
          await write(`videos/${payload}/retry`, "POST", {});
          setToast("Video queued for processing");
          await refresh();
          return;
        case "like":
          need();
          await write(`videos/${id}/like`, "POST", {
            enabled: !watchQ.data?.liked,
          });
          await refresh();
          return;
        case "subscribe":
          need();
          await write(`channels/${watchQ.data?.channel_id}/subscribe`, "POST", {
            enabled: !watchQ.data?.subscribed,
          });
          await refresh();
          return;
        case "share":
          await navigator.clipboard.writeText(location.href);
          setToast("Video link copied");
          return;
        case "save":
          need();
          show(
            "save",
            "Save to a playlist",
            [],
            [
              btn(
                "save-later",
                watchQ.data?.later ? "Remove from Watch Later" : "Watch Later",
                "outline",
                id,
              ),
              ...(playlistsQ.data || []).map((p) =>
                btn("save-to-playlist", p.title, "outline", p.id),
              ),
              btn("new-playlist", "New playlist", "primary"),
            ],
          );
          return;
        case "save-to-playlist":
          await write(`playlists/${payload}/videos`, "POST", { video_id: id });
          setDialog(null);
          setToast("Saved to playlist");
          return;
        case "save-later": {
          const v =
            payload === id ? watchQ.data : await request("videos/" + payload);
          await write(`videos/${payload}/later`, "POST", {
            enabled: !v?.later,
          });
          setDialog(null);
          await refresh();
          return;
        }
        case "video-menu": {
          const v = await request("videos/" + payload);
          show(
            "video-menu",
            v.title,
            [],
            [
              btn(
                "save-later",
                v.later ? "Remove from Watch Later" : "Watch Later",
                "outline",
                payload,
              ),
              btn("download", "Download original", "outline", payload),
              ...(v.owner === uid
                ? [
                    btn("edit-video", "Edit video", "outline", payload),
                    btn(
                      "private-access",
                      "Manage private viewers",
                      "outline",
                      payload,
                    ),
                    ...(me.drive
                      ? [
                          btn(
                            "drive-export",
                            "Export original to My Drive",
                            "outline",
                            payload,
                          ),
                        ]
                      : []),
                  ]
                : [btn("report", "Report video", "outline", payload)]),
            ],
          );
          return;
        }
        case "download":
          window.open(
            apiURL(`videos/${payload}/media/original?download=1`),
            "_blank",
            "noopener",
          );
          return;
        case "private-access":
          show(
            "access",
            "Private video access",
            [
              f("subject", "MyAccounts subject (user must have signed in)"),
              f("allow", "Access", "allow", "select", [
                { value: "allow", label: "Allow viewer" },
                { value: "revoke", label: "Revoke viewer" },
              ]),
            ],
            [btn("save-access", "Apply access", "primary", payload)],
          );
          return;
        case "save-access":
          await write(`videos/${payload}/access`, "PUT", {
            subject: form.subject,
            allow: form.allow === "allow",
          });
          setDialog(null);
          setToast("Viewer access updated");
          return;
        case "report":
          show(
            "report",
            "Report this video",
            [f("reason", "Reason", "", "textarea")],
            [btn("send-report", "Send report", "primary", payload)],
          );
          return;
        case "send-report":
          await write(`videos/${payload}/report`, "POST", {
            reason: form.reason,
          });
          setDialog(null);
          setToast("Report sent to the server administrator");
          return;
        case "reply":
          setReply(payload);
          setComment("");
          return;
        case "post-comment":
          need();
          await write(`videos/${id}/comments`, "POST", {
            body: comment,
            parent: reply,
          });
          setComment("");
          setReply("");
          await refresh();
          return;
        case "edit-comment": {
          const c = commentsQ.data?.find((c) => c.id === payload);
          show(
            "edit-comment",
            "Edit comment",
            [f("body", "Comment", c?.body, "textarea")],
            [btn("save-comment", "Save comment", "primary", payload)],
          );
          return;
        }
        case "save-comment":
          await write("comments/" + payload, "PUT", { body: form.body });
          setDialog(null);
          await refresh();
          return;
        case "delete-comment":
          await write("comments/" + payload, "DELETE");
          await refresh();
          return;
        case "seek":
          window.dispatchEvent(
            new CustomEvent("mytube-seek", { detail: Number(payload) }),
          );
          return;
        case "ended":
          if (autoplay) {
            const next = recQ.data?.find((v) => v.id !== id);
            if (next) go("watch/" + next.id);
          }
          return;
        case "playlists":
          need();
          go("playlists");
          return;
        case "open-playlist":
          setApplied("");
          setCategory("All");
          go("playlist/" + payload);
          return;
        case "new-playlist":
          need();
          show(
            "playlist",
            "Create a playlist",
            [
              f("title", "Title"),
              f("description", "Description", "", "textarea"),
              f(
                "visibility",
                "Visibility",
                "private",
                "select",
                visibilityOptions.filter(
                  (o) => me.public_uploads || o.value !== "public",
                ),
              ),
            ],
            [btn("save-playlist", "Create playlist")],
          );
          return;
        case "edit-playlist": {
          const p = await request("playlists/" + payload);
          show(
            "playlist",
            "Edit playlist",
            [
              f("title", "Title", p.title),
              f("description", "Description", p.description, "textarea"),
              f(
                "visibility",
                "Visibility",
                p.visibility,
                "select",
                visibilityOptions.filter(
                  (o) => me.public_uploads || o.value !== "public",
                ),
              ),
            ],
            [
              btn("save-playlist", "Save playlist", "primary", payload),
              btn("delete-playlist", "Delete playlist", "danger", payload),
            ],
          );
          return;
        }
        case "save-playlist":
          await write(
            "playlists" + (payload ? "/" + payload : ""),
            payload ? "PUT" : "POST",
            {
              title: form.title,
              description: form.description,
              visibility: form.visibility,
            },
          );
          setDialog(null);
          setToast("Playlist saved");
          await refresh();
          return;
        case "delete-playlist":
          await write("playlists/" + payload, "DELETE");
          setDialog(null);
          await refresh();
          return;
        case "read-notifications":
          await write("notifications/read", "POST", {});
          await refresh();
          return;
        case "clear-history":
          await write("history", "DELETE");
          setToast("Watch history cleared");
          await refresh();
          return;
        case "logout":
          await write("logout", "POST", {});
          client.clear();
          go("home");
          await refresh();
          return;
        case "admin":
          need();
          go("admin");
          return;
        case "setup": {
          const a = await request("admin");
          show(
            "setup",
            "Set up My Tube",
            [
              f("setupMode", "Authentication mode", "managed", "select", [
                { value: "managed", label: "MyAccounts (recommended)" },
                {
                  value: "local",
                  label: "Explicit local-only single-user mode",
                },
              ]),
              f("setupIssuer", "MyAccounts issuer URL", a.config.oidc_issuer),
              f("setupClient", "MyAccounts client ID", a.config.oidc_client_id),
              f(
                "setupAdmin",
                "Administrator MyAccounts subject (exact sub claim)",
                (a.config.admin_subjects || []).join(","),
              ),
              f("setupURL", "Browser-facing URL", a.config.public_url),
              f("setupMedia", "Dedicated media folder", a.config.media_dir),
              f(
                "setupUpload",
                "Maximum upload MiB",
                a.config.upload_mb,
                "number",
              ),
              f(
                "setupQuota",
                "Per-user quota MiB",
                a.config.quota_mb,
                "number",
              ),
            ],
            [btn("finish-setup", "Save setup")],
            "Register the exact URL + /auth/callback in MyAccounts. Set MYTUBE_OIDC_CLIENT_SECRET in the server environment for confidential clients.",
          );
          return;
        }
        case "finish-setup": {
          const a = await request("admin");
          await write("admin/config", "PUT", {
            ...a.config,
            mode: form.setupMode,
            oidc_issuer: form.setupIssuer,
            oidc_client_id: form.setupClient,
            admin_subjects: String(form.setupAdmin || "")
              .split(",")
              .map((s: string) => s.trim())
              .filter(Boolean),
            public_url: form.setupURL,
            media_dir: form.setupMedia,
            upload_mb: Number(form.setupUpload),
            quota_mb: Number(form.setupQuota),
          });
          setDialog(null);
          setToast(
            "Setup saved. Restart MyTube.exe to begin using your configuration.",
          );
          return;
        }
        case "save-config": {
          let value;
          try {
            value = JSON.parse(adminConfig);
          } catch {
            throw Error("Configuration must be valid JSON");
          }
          await write("admin/config", "PUT", value);
          setToast("Configuration saved. Restart MyTube.exe to apply.");
          return;
        }
        case "check-updates": {
          const update = await request("admin/update");
          setToast(
            update.latest
              ? `Current ${update.current} · latest published ${update.latest}${update.prerelease ? " (preview)" : ""}. Download updates from Releases in Settings.`
              : "No published release is available yet.",
          );
          return;
        }
        case "reconcile": {
          const result = await write("admin/reconcile", "POST", {
            purge_deleted: false,
          });
          setToast(
            `Storage checked. ${result.missing.length} missing originals.`,
          );
          await refresh();
          return;
        }
        case "manage-user": {
          const u = adminQ.data?.users.find((u: any) => u.id === payload);
          show(
            "user",
            u.name,
            [
              f("quota", "Quota in MiB", u.quota_mb, "number"),
              f(
                "disabled",
                "Account status",
                u.disabled ? "disabled" : "enabled",
                "select",
                ["enabled", "disabled"],
              ),
            ],
            [btn("save-user", "Save account", "primary", payload)],
          );
          return;
        }
        case "save-user":
          await write("admin/users/" + payload, "PUT", {
            disabled: form.disabled === "disabled",
            quota_mb: Number(form.quota),
          });
          setDialog(null);
          await refresh();
          return;
        case "resolve-report":
        case "remove-reported":
          await write("admin/reports/" + payload, "PUT", {
            status: "resolved",
            remove: name === "remove-reported",
          });
          await refresh();
          return;
        case "drive":
        case "drive-export":
          need();
          show(
            name,
            name === "drive" ? "Import from My Drive" : "Export to My Drive",
            [
              f("driveSource", "Drive source name"),
              f(
                "drivePath",
                name === "drive"
                  ? "Scoped file or directory path"
                  : "New destination file path",
                "/",
              ),
              f(
                "driveToken",
                "Your scoped Drive token (optional on Workspace origin)",
                "",
                "password",
              ),
              ...(name === "drive"
                ? [
                    f(
                      "driveSize",
                      "File size in bytes (if Drive omits Content-Length)",
                      "",
                      "number",
                    ),
                  ]
                : []),
            ],
            [
              ...(name === "drive"
                ? [btn("drive-list", "Browse directory", "outline")]
                : []),
              btn(
                name === "drive"
                  ? "drive-import-confirm"
                  : "drive-export-confirm",
                name === "drive" ? "Import selected file" : "Export original",
                "primary",
                payload,
              ),
            ],
            "Requests use your own Drive session or scoped token. Administrator tokens are rejected. Export never overwrites an existing file.",
          );
          return;
        case "drive-list": {
          const listing = await write(
            "drive/list",
            "POST",
            { source: form.driveSource, path: form.drivePath },
            { "X-Drive-Token": form.driveToken || "" },
          );
          const items = listing.items || [
            ...(listing.folders || []),
            ...(listing.files || []),
          ];
          setDialog((old: any) => ({
            ...old,
            actions: [
              ...items.slice(0, 200).map((item: any) =>
                btn(
                  "drive-select",
                  `${item.type === "directory" ? "Folder: " : "Video: "}${item.name}`,
                  "outline",
                  {
                    path:
                      item.path ||
                      String(form.drivePath).replace(/\/$/, "") +
                        "/" +
                        item.name,
                    size: item.size,
                    isDir: item.type === "directory",
                  },
                ),
              ),
              btn("drive-list", "Browse directory", "outline"),
              btn("drive-import-confirm", "Import selected file"),
            ],
            message: `${listing.path || form.drivePath}: ${items.map((v: any) => v.name + (v.type === "directory" ? "/" : "")).join(", ") || "Empty directory"}`,
          }));
          return;
        }
        case "drive-select":
          setForm((old: any) => ({
            ...old,
            drivePath: payload.path,
            driveSize: String(payload.size || ""),
          }));
          setDialog((old: any) => ({
            ...old,
            message: payload.isDir
              ? "Folder selected. Choose Browse directory to open it."
              : "File selected. Choose Import selected file to create a private copy.",
          }));
          return;
        case "drive-import-confirm": {
          const result = await write(
            "drive/import",
            "POST",
            {
              source: form.driveSource,
              path: form.drivePath,
              size: Number(form.driveSize || 0),
              visibility: "private",
            },
            { "X-Drive-Token": form.driveToken || "" },
          );
          setDialog(null);
          await refresh();
          go("watch/" + result.id);
          return;
        }
        case "drive-export-confirm":
          await write(
            `videos/${payload}/drive-export`,
            "POST",
            { source: form.driveSource, path: form.drivePath },
            { "X-Drive-Token": form.driveToken || "" },
          );
          setDialog(null);
          setToast("Original exported to My Drive");
          return;
      }
    } catch (e: any) {
      setError(e.message || "The action failed");
    }
  }
  function field(name: string, value: any) {
    switch (name) {
      case "search":
        setSearch(value);
        break;
      case "sort":
        setSort(value);
        break;
      case "duration":
        setDur(value);
        break;
      case "days":
        setDays(value);
        break;
      case "autoplay":
        setAutoplay(value);
        break;
      case "comment":
        setComment(value);
        break;
      case "adminConfig":
        setAdminConfig(value);
        break;
      case "uploadVisibility":
        setUploadVisibility(value);
        break;
      case "personalize":
        client.setQueryData(["me"], (old: any) => ({
          ...old,
          personalize: value,
        }));
        void write("preferences", "PUT", { personalize: value })
          .then(refresh)
          .catch((e) => {
            setError(e.message);
            void refresh();
          });
        break;
      case "files":
        for (const file of value) {
          const pending = tasks.find(
            (u) => u.name === file.name && u.size === file.size && !u.video_id,
          );
          void upload(file, pending);
        }
        break;
      default:
        setForm((old: any) => ({ ...old, [name]: value }));
    }
  }
  const videos = (videosQ.data || []).map(decorate);
  const v = watchQ.data ? decorate(watchQ.data) : {};
  const c = channelQ.data || {};
  const titles: any = {
    home: applied ? `Results for “${applied}”` : "Find your next favourite",
    subscriptions: "From the channels you follow",
    shorts: "Little stories. Big ideas.",
    history: "Pick up where you left off",
    later: "Saved for a quieter moment",
    likes: "Videos you liked",
    explore: "Explore the library",
    playlist: "Your curated collection",
  };
  const navItems = [
    ["home", "House", "Home"],
    ["shorts", "Flame", "Shorts"],
    ["subscriptions", "TvMinimalPlay", "Subscriptions"],
    ["explore", "Compass", "Explore"],
  ];
  const libraryItems = [
    ["history", "History", "History"],
    ["later", "Clock3", "Watch Later"],
    ["likes", "ThumbsUp", "Liked videos"],
    ["studio", "Video", "Your videos"],
  ];
  const nav = (list: string[][]) =>
    list.map(([key, icon, label]) => ({
      key,
      icon,
      label,
      variant: screen === key ? "nav active" : "nav",
    }));
  const queries = [
    meQ,
    videosQ,
    screen === "watch" ? watchQ : null,
    screen === "channel" ? channelQ : null,
    screen === "admin" ? adminQ : null,
    screen === "imports" ? importsQ : null,
  ];
  const queryError = queries.find((q) => q?.error)?.error as Error | undefined;
  return {
    action,
    field,
    app: {
      version: me.version || "My Tube",
      csrf,
      signedIn,
      user: me.user,
      initials: initials(me.user?.name),
      avatar: String(me.user?.avatar || "").startsWith("https://")
        ? me.user.avatar
        : "",
      loginURL: base + "auth/login",
      shellRegion: menu ? "shell" : "shell collapsed",
      nav: nav(navItems),
      libraryNav: nav(libraryItems),
      error: error || queryError?.message,
      toast,
      loading: meQ.isLoading,
      search,
      heading: titles[screen] || "Your library",
      subheading: applied
        ? "Search respects each video’s visibility."
        : "Independent creators. Shared interests. A library that’s yours.",
      setupScreen: me.setup && signedIn,
      homeScreen:
        !(me.setup && signedIn) &&
        [
          "home",
          "shorts",
          "subscriptions",
          "history",
          "later",
          "likes",
          "explore",
          "playlist",
        ].includes(screen),
      showFilters,
      sort,
      days,
      duration: dur,
      sortOptions: [
        { value: "recent", label: "Recently uploaded" },
        { value: "popular", label: "Most viewed" },
        { value: "oldest", label: "Oldest first" },
      ],
      dateOptions: [
        { value: "", label: "Any time" },
        { value: "1", label: "Today" },
        { value: "7", label: "This week" },
        { value: "30", label: "This month" },
      ],
      durationOptions: [
        { value: "", label: "Any duration" },
        { value: "short", label: "Under 4 minutes" },
        { value: "long", label: "Over 20 minutes" },
      ],
      categories: [
        "All",
        "Music",
        "Learning",
        "Technology",
        "Travel",
        "Gaming",
        "Film",
        "Nature",
        "General",
      ].map((name) => ({
        name,
        variant: category === name ? "chip active" : "chip",
      })),
      videos,
      hasVideos: videos.length > 0,
      emptyTitle: signedIn
        ? "Your next favourite starts here"
        : "Welcome to your own corner of video",
      emptyBody: signedIn
        ? "Upload your first video, follow a creator, or try another filter."
        : "Sign in with MyAccounts to explore your community and share a video.",
      watchScreen: screen === "watch" && !!watchQ.data,
      watchRegion: cinema ? "watch-layout cinema" : "watch-layout",
      video: v,
      autoplay,
      likeLabel: `${v.likes || 0}`,
      likeVariant: v.liked ? "pill active" : "pill",
      subscribeIntent: v.owner === uid ? "edit-video" : "subscribe",
      subscribeLabel:
        v.owner === uid
          ? "Manage video"
          : v.subscribed
            ? "Subscribed"
            : "Subscribe",
      subscribeVariant: v.subscribed ? "outline" : "primary",
      comment,
      commentPlaceholder: reply ? "Write a reply…" : "Add to the conversation…",
      commentHeading: `${commentsQ.data?.length || 0} comments`,
      comments: (commentsQ.data || []).map((c) => ({
        ...c,
        initials: initials(c.name),
        date: relative(c.created),
        region: c.parent ? "comment reply" : "comment",
        canEdit: c.user_id === uid,
        canDelete: c.user_id === uid || v.owner === uid,
      })),
      recommended: (recQ.data || [])
        .filter((x) => x.id !== id)
        .slice(0, 8)
        .map(decorate),
      chapters: (() => {
        try {
          return (
            JSON.parse(v.chapters || "[]")?.map((c: any) => ({
              ...c,
              label: `${duration(c.time)}  ${c.title}`,
            })) || []
          );
        } catch {
          return [];
        }
      })(),
      studioScreen: screen === "studio",
      analytics: analyticsQ.data || {},
      watchTime: `${((analyticsQ.data?.watch_seconds || 0) / 3600).toFixed(1)}h`,
      ownedVideos: screen === "studio" ? videos : [],
      channelScreen: screen === "channel" && !!channelQ.data,
      channelData: c,
      channelBanner: c.id ? apiURL(`channels/${c.id}/image/banner`) : "",
      channelInitials: initials(c.name),
      channelAvatar: c.id ? apiURL(`channels/${c.id}/image/avatar`) : "",
      channelDetails: c.imported
        ? `${c.handle ? "@" + c.handle + " · " : ""}Imported channel · Private · 480p downloads`
        : `@${c.handle || ""} · ${c.subscribers || 0} subscribers`,
      ownChannel: c.owner === uid && !c.imported,
      importedChannel: !!c.imported,
      showImportedChannelPending:
        !!c.imported && channelTab !== "about" && channelTab !== "playlists",
      importedChannels: (channelsQ.data || []).filter(
        (channel: any) => channel.imported,
      ),
      importedChannelBusy: [
        "queued",
        "preparing",
        "scanning",
        "downloading",
        "cancelling",
      ].includes(c.import_state),
      importedChannelPending: (c.items || [])
        .filter((i: any) => i.state !== "saved")
        .map((i: any) => ({
          ...i,
          label:
            (
              {
                queued: "Waiting to download",
                downloading: "Downloading at up to 480p",
                saving: "Verifying and saving",
                failed: "Download failed",
                interrupted: "Waiting for retry",
              } as any
            )[i.state] || i.state,
        })),
      channelAbout: channelTab === "about",
      channelPlaylists: channelTab === "playlists",
      channelLists: (channelListsQ.data || []).filter(
        (p: any) => !c.imported || p.id === c.playlist_id,
      ),
      channelVideos: videos,
      playlistsScreen: screen === "playlists",
      playlists: (playlistsQ.data || []).map((p) => ({
        ...p,
        subtitle: `${p.count} videos · ${p.visibility}`,
      })),
      notificationsScreen: screen === "notifications",
      notifications: notificationsQ.data || [],
      settingsScreen: screen === "settings",
      accountLabel: signedIn
        ? `${me.user.name} · ${me.mode === "local" ? "Local owner" : "MyAccounts"}`
        : "Signed out",
      personalize: me.personalize,
      streamingStatus: me.streaming?.message
        ? me.streaming.message +
          (me.streaming.total && me.streaming.phase === "downloading"
            ? ` ${Math.min(100, Math.floor((100 * me.streaming.downloaded) / me.streaming.total))}% (${(me.streaming.downloaded / 1048576).toFixed(1)} / ${(me.streaming.total / 1048576).toFixed(1)} MiB)`
            : "")
        : "Preparing adaptive video quality…",
      playbackStatus: watchQ.data?.hls
        ? "Adaptive quality is ready. Auto adjusts playback to your connection."
        : watchQ.data?.status === "failed"
          ? "Video optimisation needs attention. The original remains playable; retry processing in Creator Studio."
          : me.streaming?.phase === "error"
            ? me.streaming.message
            : me.transcode
              ? "Preparing smaller video qualities. The original remains playable and may need more bandwidth until processing finishes."
              : "Playing the original file. Automatic video optimisation is disabled.",
      helperStatus: `FFprobe: ${me.ffprobe ? "available" : "not configured"} · FFmpeg: ${me.ffmpeg ? "available" : "not configured"}`,
      driveEnabled: me.drive,
      isAdmin: me.user?.admin,
      adminScreen: screen === "admin" && me.user?.admin,
      adminConfig,
      adminSummary: `${((adminQ.data?.media_bytes || 0) / 1073741824).toFixed(2)} GiB stored · ${adminQ.data?.storage_health || "checking storage"}`,
      adminUsers: (adminQ.data?.users || []).map((u: any) => ({
        ...u,
        quotaLabel: `${u.quota_mb} MiB · ${u.disabled ? "disabled" : "enabled"}`,
      })),
      jobs: adminQ.data?.jobs || [],
      reports: adminQ.data?.reports || [],
      logs: (adminQ.data?.logs || []).map((l: any) => ({
        ...l,
        label: `${new Date(l.at).toLocaleString()} · ${l.event} · ${l.target}`,
      })),
      uploadScreen: screen === "upload",
      importsScreen: screen === "imports",
      importsLoading: importsQ.isLoading,
      hasImports: !!importsQ.data?.length,
      importBusy,
      imports: (importsQ.data || []).map((s: any) => {
        const busy = [
          "queued",
          "preparing",
          "scanning",
          "downloading",
          "cancelling",
        ].includes(s.state);
        const items = s.items || [];
        const saved = items.filter((i: any) => i.state === "saved").length;
        const failed = items.filter((i: any) => i.state === "failed").length;
        return {
          ...s,
          busy,
          canCancel: busy && s.state !== "cancelling",
          canCheck: !busy,
          canRemove: !busy,
          stateLabel:
            (
              {
                queued: "Waiting",
                preparing: "Preparing tools",
                scanning: "Checking YouTube",
                downloading: "Importing",
                completed: "Up to date",
                partial: "Some videos need attention",
                failed: "Needs attention",
                cancelling: "Cancelling",
                cancelled: "Cancelled",
              } as any
            )[s.state] || s.state,
          summary: `${s.kind === "channel" ? "Channel · newest uploads" : `Playlist · ${s.playlist_order === "last" ? "end" : "start"} of playlist`} · 480p · up to ${s.latest} videos per check · ${s.auto_hours ? "daily checks" : "manual checks"}`,
          counts: `Recent results: ${saved} saved · ${failed} need attention`,
          timing:
            busy && s.started
              ? `Started ${new Date(s.started).toLocaleTimeString()}. You can leave this page; imports continue on the server.`
              : s.finished
                ? `Last finished ${new Date(s.finished).toLocaleString()}${s.auto_hours ? ` · Next check ${new Date(s.next_check).toLocaleString()}` : ""}`
                : "Waiting for the server to start this import.",
          items: items.map((i: any) => ({
            ...i,
            active: i.state === "downloading",
            playable: i.state === "saved" && !!i.video_id,
            percent:
              i.total > 0
                ? Math.min(99, Math.floor((100 * i.bytes) / i.total))
                : null,
            progressLabel:
              i.state === "saving"
                ? "Downloaded · verifying and saving…"
                : i.state === "downloading"
                  ? `${(i.bytes / 1048576).toFixed(1)} MiB downloaded${i.total > 0 ? ` of ${(i.total / 1048576).toFixed(1)} MiB` : " · total size not supplied by YouTube"}`
                  : i.message,
          })),
        };
      }),
      uploadHelp: `Up to ${me.upload_mb || 2048} MiB per file. Uploads can resume after an interruption.`,
      uploadTasks: tasks,
      uploadVisibility,
      visibilityOptions: visibilityOptions.filter(
        (o) => me.public_uploads || o.value !== "public",
      ),
      dialog,
      dialogTitle: dialog?.title,
      dialogMessage: dialog?.message,
      dialogFields: (dialog?.fields || []).map((f: any) => ({
        ...f,
        value: form[f.key],
      })),
      dialogActions: (dialog?.actions || []).map((a: any) => ({
        ...a,
        disabled: a.intent.startsWith("youtube-") && importBusy,
      })),
    },
  };
}
