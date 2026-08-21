class SquadOc < Formula
  desc "Human-led AI agent teams for OpenCode"
  homepage "https://github.com/xeaser/squad-opencode"
  version "0.4.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/xeaser/squad-opencode/releases/download/v0.4.0/squad-oc_0.4.0_darwin_arm64.tar.gz"
      sha256 "e2c662e3895a7b937aa0539b354fc6705dd56fdb833cd7222d63d07d4afb546b"
    else
      url "https://github.com/xeaser/squad-opencode/releases/download/v0.4.0/squad-oc_0.4.0_darwin_amd64.tar.gz"
      sha256 "ee6f9417b71e96904c8522299f8243d46a785b724b66d31c869051d734fd7777"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/xeaser/squad-opencode/releases/download/v0.4.0/squad-oc_0.4.0_linux_arm64.tar.gz"
      sha256 "e8fa3657fb9b270fc87ef3f24c2236fa751664597245d2976124541933abaf21"
    else
      url "https://github.com/xeaser/squad-opencode/releases/download/v0.4.0/squad-oc_0.4.0_linux_amd64.tar.gz"
      sha256 "c54df6f427bc97b3fbf7a337fe9806e0a29bc69864f9c36133879d24e65a7c0a"
    end
  end

  def install
    bin.install "squad-oc"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/squad-oc version")
  end
end
