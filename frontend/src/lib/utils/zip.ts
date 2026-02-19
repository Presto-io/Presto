// Minimal ZIP creator for packaging PDF results client-side.
// Uses "stored" method (no compression) since PDFs are already compressed.

const crc32Table = new Uint32Array(256);
for (let i = 0; i < 256; i++) {
  let c = i;
  for (let j = 0; j < 8; j++) {
    c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
  }
  crc32Table[i] = c;
}

function crc32(data: Uint8Array): number {
  let crc = 0xffffffff;
  for (let i = 0; i < data.length; i++) {
    crc = crc32Table[(crc ^ data[i]) & 0xff] ^ (crc >>> 8);
  }
  return (crc ^ 0xffffffff) >>> 0;
}

export function createZip(files: { name: string; data: Uint8Array }[]): Blob {
  const enc = new TextEncoder();
  const entries: { nameB: Uint8Array; crc: number; size: number; offset: number }[] = [];
  const chunks: Uint8Array[] = [];
  let offset = 0;

  for (const f of files) {
    const nameB = enc.encode(f.name);
    const c = crc32(f.data);

    // Local file header (30 bytes)
    const lh = new DataView(new ArrayBuffer(30));
    lh.setUint32(0, 0x04034b50, true); // signature
    lh.setUint16(4, 20, true); // version needed
    lh.setUint16(6, 0, true); // flags
    lh.setUint16(8, 0, true); // compression: stored
    lh.setUint16(10, 0, true); // mod time
    lh.setUint16(12, 0, true); // mod date
    lh.setUint32(14, c, true); // crc32
    lh.setUint32(18, f.data.length, true); // compressed size
    lh.setUint32(22, f.data.length, true); // uncompressed size
    lh.setUint16(26, nameB.length, true); // filename length
    lh.setUint16(28, 0, true); // extra field length

    entries.push({ nameB, crc: c, size: f.data.length, offset });
    chunks.push(new Uint8Array(lh.buffer), nameB, f.data);
    offset += 30 + nameB.length + f.data.length;
  }

  // Central directory
  const cdStart = offset;
  for (const e of entries) {
    const cd = new DataView(new ArrayBuffer(46));
    cd.setUint32(0, 0x02014b50, true); // signature
    cd.setUint16(4, 20, true); // version made by
    cd.setUint16(6, 20, true); // version needed
    cd.setUint16(8, 0, true); // flags
    cd.setUint16(10, 0, true); // compression: stored
    cd.setUint16(12, 0, true); // mod time
    cd.setUint16(14, 0, true); // mod date
    cd.setUint32(16, e.crc, true); // crc32
    cd.setUint32(20, e.size, true); // compressed size
    cd.setUint32(24, e.size, true); // uncompressed size
    cd.setUint16(28, e.nameB.length, true); // filename length
    cd.setUint16(30, 0, true); // extra field length
    cd.setUint16(32, 0, true); // comment length
    cd.setUint16(34, 0, true); // disk number start
    cd.setUint16(36, 0, true); // internal attributes
    cd.setUint32(38, 0, true); // external attributes
    cd.setUint32(42, e.offset, true); // local header offset

    chunks.push(new Uint8Array(cd.buffer), e.nameB);
    offset += 46 + e.nameB.length;
  }

  // End of central directory (22 bytes)
  const cdSize = offset - cdStart;
  const eocd = new DataView(new ArrayBuffer(22));
  eocd.setUint32(0, 0x06054b50, true); // signature
  eocd.setUint16(4, 0, true); // disk number
  eocd.setUint16(6, 0, true); // disk with CD
  eocd.setUint16(8, entries.length, true); // entries on this disk
  eocd.setUint16(10, entries.length, true); // total entries
  eocd.setUint32(12, cdSize, true); // CD size
  eocd.setUint32(16, cdStart, true); // CD offset
  eocd.setUint16(20, 0, true); // comment length

  chunks.push(new Uint8Array(eocd.buffer));
  return new Blob(chunks as BlobPart[], { type: 'application/zip' });
}
