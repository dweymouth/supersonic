# Usage

The Dockerfile will take care of installing all needed dependencies; all you should need to do once you enter the container is run `make` and then `./supersonic` to start it.  See the Linux/Ubuntu sections of BUILD.md for more details.  

# VNC Access

The dev container by default will open a desktop accessible via VNC at port :5901 or via web browser at :6080.

You can use the VNC_RESOLUTION env var to adjust it if your monitors don't fit 1080p well

supersonic can still be started via the terminal or debugger in VS Code; it will automatically open in the VNC desktop


# Sound

The container is configured to forward the default `/dev/snd` card so that supersonic can access it via the default ALSA driver.

Depending on your host system, you may need to select a different sound card than what your system is using, or configure a dummy sound card, etc, as certain audio systems require exclusive access to the hardware device and don't like to share it with the container.

If you encounter permission issues with accessing the device, make sure your user is in the `audio` group

# Config

The default config (`default-config.toml`) will be copied to the proper location in the home directory.  If you create a file in `.devcontainer/` named `custom-config.toml`, that will be copied instead.  It's in the `.gitignore`, so you can safely put whatever changes you want in there to point at your chosen servers, etc and persist them across dev container rebuilds.  

# Future Improvements

- Better sound integration to enable mixing with host system
- X11 forwarding instructions