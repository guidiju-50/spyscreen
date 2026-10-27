(function () {
  const report = {
    timestamp: new Date().toISOString(),
    userAgent: navigator.userAgent,
    platform: navigator.platform,
    mediaDevices: typeof navigator.mediaDevices !== 'undefined',
    getDisplayMedia:
      typeof navigator.mediaDevices !== 'undefined' &&
      typeof navigator.mediaDevices.getDisplayMedia === 'function',
    note: 'This page does not detect all capture methods. Phase 3 adds structured A/B/C scenarios.',
  };
  console.log('diagnostics baseline', report);
})();
