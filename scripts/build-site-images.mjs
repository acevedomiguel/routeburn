// Converts the VHS terminal capture into the deployed AVIF/WebP pair.
// Run from the repository root: node scripts/build-site-images.mjs
import sharp from 'sharp';

const source = 'assets-src/terminal-demo.png';
const targets = [
  ['site/assets/images/terminal-demo.avif', (img) => img.avif({ quality: 60 })],
  ['site/assets/images/terminal-demo.webp', (img) => img.webp({ quality: 85 })],
];

const base = sharp(source);
const { width, height } = await base.metadata();
if (!width || !height) {
  throw new Error(`could not read dimensions of ${source}`);
}

for (const [target, encode] of targets) {
  await encode(base.clone()).toFile(target);
  console.log(`wrote ${target} (${width}x${height})`);
}
