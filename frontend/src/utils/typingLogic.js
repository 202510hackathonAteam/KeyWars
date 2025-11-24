import { romanMap } from "./romajiMap";

/**
 * タイピング判定ロジック
 * @param {string} input 入力欄の全テキスト
 * @param {string} target target_romaji ("komorebi" など)
 * @param {number} progress いま何文字消費しているか
 * @returns {object} { newProgress, isCorrect, isFinish, miss }
 */
export function checkTyping(input, target, progress) {
  const miss = 0;

  // 入力された最新の文字
  const last = input[input.length - 1];
  if (!last) return { newProgress: progress, isCorrect: false, isFinish: false, miss };

  // target の次に必要な文字列
  const remaining = target.slice(progress);

  // (1) 1文字一致チェック
  if (remaining.startsWith(last)) {
    const newProgress = progress + 1;
    const isFinish = newProgress === target.length;
    return { newProgress, isCorrect: true, isFinish, miss };
  }

  // (2) 特殊：しゃ / でぃ / きょ など辞書による複数パターン
  for (const [kana, patterns] of Object.entries(romanMap)) {
    for (const p of patterns) {
      if (remaining.startsWith(p)) {
        // ユーザーが "p" を一文字ずつ入力している途中
        // 例: s → sh → sha
        const typed = input.slice(-p.length);
        if (p.startsWith(typed)) {
          // まだ完成ではないが、間違っていない
          return { newProgress: progress, isCorrect: true, isFinish: false, miss };
        }

        // 完全一致
        if (typed === p) {
          const newProgress = progress + p.length;
          const isFinish = newProgress === target.length;
          return { newProgress, isCorrect: true, isFinish, miss };
        }
      }
    }
  }

  // (3) 不一致 → ミス
  return { newProgress: progress, isCorrect: false, isFinish: false, miss: 1 };
}
