class SquadOc < Formula
  desc "Human-led AI agent teams for OpenCode"
  homepage "https://github.com/xeaser/squad-opencode"
  version "0.5.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/xeaser/squad-opencode/releases/download/v0.5.0/squad-oc_0.5.0_darwin_arm64.tar.gz"
      sha256 "c1deeb0932e3cbaaee08a92c728845301214af322a2dca01c2d455a9f8ef604e"
    else
      url "https://github.com/xeaser/squad-opencode/releases/download/v0.5.0/squad-oc_0.5.0_darwin_amd64.tar.gz"
      sha256 "6005c0a55c65edc9c1a7bdde7cc5ebf4668c5097e36f9febb06b705d90dae254"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/xeaser/squad-opencode/releases/download/v0.5.0/squad-oc_0.5.0_linux_arm64.tar.gz"
      sha256 "190803966a61fae6a98729e712ff664e38a1e5189c30f40e3031b1ecbb347ad2"
    else
      url "https://github.com/xeaser/squad-opencode/releases/download/v0.5.0/squad-oc_0.5.0_linux_amd64.tar.gz"
      sha256 "94dce0b564c81a6e90b1e367cbdee40cfb29fa43a484fe3ac799e61a34f9c869"
    end
  end

  def install
    bin.install "squad-oc"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/squad-oc version")
  end
end
