class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.4"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.4/avdslim_v1.0.4_darwin_arm64.tar.gz"
      sha256 "b5b7a834c9cd827154f1201121f02133c858ce7075072619a515a4ad61e43b60"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.4/avdslim_v1.0.4_darwin_amd64.tar.gz"
      sha256 "26ff908fc719c5aba47056bfc0c26272feb55c3966dc22743787644fa2449738"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.4/avdslim_v1.0.4_linux_arm64.tar.gz"
      sha256 "905dd7596415950095cd0fec076280948d47d06f13b96e0063c9f1d4df62f098"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.4/avdslim_v1.0.4_linux_amd64.tar.gz"
      sha256 "bc418d66d5cd6d565a9d8b3b144cc491d76b3052b320544e5aee30dab910b7ea"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "AVD-SLIM", shell_output("#{bin}/avdslim version")
  end
end
