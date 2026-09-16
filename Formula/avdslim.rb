class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.6"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.6/avdslim_v1.0.6_darwin_arm64.tar.gz"
      sha256 "49ab1ef5034b1077ac8e612af26345809a91ee160fe69e251f9faa2dafb5725c"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.6/avdslim_v1.0.6_darwin_amd64.tar.gz"
      sha256 "dd0c67cccea0c648af3f135c3bbaa1340675939f783b3ac996f500c2ba21bc53"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.6/avdslim_v1.0.6_linux_arm64.tar.gz"
      sha256 "2cdc54f60daebf9061f74fb79802464bb99b4cac2831d59e6d618f67a3335503"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.6/avdslim_v1.0.6_linux_amd64.tar.gz"
      sha256 "96f260edfffb2bbe13cb284d2bcd66411e722f47f4091834cea9bc0c26d8a05d"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "avdslim", shell_output("#{bin}/avdslim version")
  end
end
