// src/utils/kanaToRomaji.js
import { romanMap } from "./romajiMap";

/**
 * ひらがな → ローマ字（最も基本の表記）に変換する。
 * 例: "なつのもののけ" → "natsunomononoke"
 */
export function kanaToStrictRomaji(kanaText) {
  let romaji = "";
  let i = 0;

  while (i < kanaText.length) {
    // 2文字の拗音（きゃ、しゃ、ちゃ、など）
    const two = kanaText.slice(i, i + 2);
    if (romanMap[two]) {
      romaji += romanMap[two][0]; // 最初の候補を採用
      i += 2;
      continue;
    }

    // 1文字（か・さ・た など）
    const one = kanaText[i];
    if (romanMap[one]) {
      romaji += romanMap[one][0];
      i += 1;
      continue;
    }

    // 未対応文字があればそのまま進む（ほぼ起きない）
    romaji += one;
    i += 1;
  }

  return romaji;
}
