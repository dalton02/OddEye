import { filesOfFormat, fmtBytes, formatsOf, percent, torrentTrack, trackKey, trackURL, folderTrack, type TorrentFile } from "../src/lib/music";

let failed = 0;
function check(name: string, got: unknown, want: unknown): void {
  const g = JSON.stringify(got);
  const w = JSON.stringify(want);
  if (g !== w) {
    failed++;
    console.error(`${name}: ${g}; want ${w}`);
  }
}

const hash = "a".repeat(40);
check("trackURL torrent", trackURL(torrentTrack(hash, "cd 1/01 #intro.mp3")), `/torrent/${hash}/cd%201%2F01%20%23intro.mp3`);
check("trackURL pasta", trackURL(folderTrack("minha música.ogg")), "/music/minha%20m%C3%BAsica.ogg");
check("título", torrentTrack(hash, "cd1/02 - Song.flac").title, "02 - Song");
check("key", trackKey(torrentTrack(hash.toUpperCase(), "x.mp3")), `t:${hash}:x.mp3`);

const f = (path: string): TorrentFile => ({ path, size: 1, done: 0, complete: false, wanted: false });
const files = [f("10 fim.flac"), f("2 meio.mp3"), f("10 fim.mp3"), f("1 inicio.mp3"), f("1 inicio_64kb.mp3"), f("so64_64kb.mp3"), f("capa.ogg"), f("x.xyz")];
check("formatos", formatsOf(files), ["mp3", "ogg", "flac", "xyz"]);
check(
  "mp3 sem cópias de 64kb repetidas, em ordem numérica",
  filesOfFormat(files, "mp3").map((x) => x.path),
  ["1 inicio.mp3", "2 meio.mp3", "10 fim.mp3", "so64_64kb.mp3"]
);
check("flac", filesOfFormat(files, "flac").map((x) => x.path), ["10 fim.flac"]);

check("bytes", [fmtBytes(512), fmtBytes(1536), fmtBytes(5 * 1024 * 1024), fmtBytes(300 * 1024 * 1024)], ["512 B", "1.5 KB", "5.0 MB", "300 MB"]);
check("percent", [percent(0, 0), percent(1, 3), percent(5, 4)], [0, 33, 100]);

if (failed) {
  console.error(`${failed} falha(s)`);
  process.exit(1);
}
console.log("music: ok");
