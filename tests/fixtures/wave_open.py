import wave
import os
import tempfile


def le32(n: int) -> str:
    return chr(n & 0xFF) + chr((n >> 8) & 0xFF) + chr((n >> 16) & 0xFF) + chr((n >> 24) & 0xFF)


def le16(n: int) -> str:
    return chr(n & 0xFF) + chr((n >> 8) & 0xFF)


def main() -> None:
    d = tempfile.mkdtemp()
    path = os.path.join(d, "tone.wav")
    nframes = 4
    nch = 1
    sampwidth = 1
    framerate = 8000
    byte_rate = framerate * nch * sampwidth
    block_align = nch * sampwidth
    data_size = nframes * block_align

    fmt = "fmt " + le32(16) + le16(1) + le16(nch) + le32(framerate) + le32(byte_rate) + le16(block_align) + le16(sampwidth * 8)
    data = "data" + le32(data_size) + chr(1) + chr(2) + chr(3) + chr(4)
    riff_size = 4 + len(fmt) + len(data)
    riff = "RIFF" + le32(riff_size) + "WAVE"
    header = riff + fmt + data
    with open(path, "w") as fh:
        fh.write(header)

    with wave.open(path, "rb") as wf:
        print(wf.getnchannels())
        print(wf.getsampwidth())
        print(wf.getframerate())
        print(wf.getnframes())
        frames = wf.readframes(2)
        print(len(frames))


if __name__ == "__main__":
    main()
