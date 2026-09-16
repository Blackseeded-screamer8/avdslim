class Avdslim < Formula
  desc "Drop Android Virtual Device (AVD) RAM from ~8GB to ~1.5GB on Apple Silicon & Linux"
  homepage "https://github.com/kdbhalala/avdslim"
  version "1.0.5"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.5/avdslim_v1.0.5_darwin_arm64.tar.gz"
      sha256 "c82c5d13744d9b02f9a3a93a5f761253586f16ad0be313c4365ade0ed76497f5"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.5/avdslim_v1.0.5_darwin_amd64.tar.gz"
      sha256 "266b1a19cb5899ef73de6ccdfd96058de4c1355801f272b537469614995909c3"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.5/avdslim_v1.0.5_linux_arm64.tar.gz"
      sha256 "f4e750ad1a657448eb83c9b872fff53bd6ae2d9ceb8f420f9309560aa9256f5d"
    else
      url "https://github.com/kdbhalala/avdslim/releases/download/v1.0.5/avdslim_v1.0.5_linux_amd64.tar.gz"
      sha256 "a893b316a00b483bf9ba292345e562537a0eb6566ee700893e66ea47f6160465"
    end
  end

  def install
    bin.install "avdslim"
  end

  test do
    assert_match "avdslim", shell_output("#{bin}/avdslim version")
  end
end
