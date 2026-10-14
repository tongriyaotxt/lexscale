'use client';

import { useState } from 'react';

export function useDarkMode() {
  const [isDark, setIsDark] = useState(false);

  const toggle = () => {
    const next = !isDark;
    setIsDark(next);
    document.documentElement.classList.toggle('dark', next);
  };

  return { isDark, toggle };
}
