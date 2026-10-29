# Third-party Runtime Notices

The Windows guided package bundles the pinned `whisper.cpp` Windows x64
runtime archive and the multilingual Whisper `base` model. It does not bundle
FFmpeg/FFprobe; those are downloaded from the pinned manifest source only after
the user sees the asset details and grants consent.

## whisper.cpp and ggml runtime

- Version: whisper.cpp 1.9.5, source commit
  `d1be6fde11ac6e0407606b4e42fe72d34add8037`.
- Binary archive: `whisper-bin-x64.zip`, build `b5454`, SHA-256
  `6ba69e3482d7826214f90a6a9c84ca07782aec1e1d0c6a7c30c994fd5d816ccb`.
- Source and license: <https://github.com/ggml-org/whisper.cpp/tree/v1.9.5>.
- The archive includes SDL2 2.28.5 (`SDL2.dll`), whose license is reproduced
  below.

### MIT License — whisper.cpp and ggml

Copyright (c) 2023-2026 The ggml authors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## OpenAI Whisper `base` model

- Model: multilingual `ggml-base.bin`, repository commit
  `5359861c739e955e79d9a303bcbc70fb988958b1`, SHA-256
  `60ed5bc3dd14eea856493d334349b405782ddcaf0028d4b5df4088345fba2efe`.
- Source and license: <https://github.com/openai/whisper/blob/main/LICENSE>.

### MIT License — OpenAI Whisper model

Copyright (c) 2022 OpenAI

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## SDL2 2.28.5

Source: <https://github.com/libsdl-org/SDL/tree/release-2.28.x>.

Copyright (C) 1997-2023 Sam Lantinga <slouken@libsdl.org>

This software is provided 'as-is', without any express or implied
warranty. In no event will the authors be held liable for any damages
arising from the use of this software.

Permission is granted to anyone to use this software for any purpose,
including commercial applications, and to alter it and redistribute it
freely, subject to the following restrictions:

1. The origin of this software must not be misrepresented; you must not
   claim that you wrote the original software. If you use this software
   in a product, an acknowledgment in the product documentation would be
   appreciated but is not required.
2. Altered source versions must be plainly marked as such, and must not be
   misrepresented as being the original software.
3. This notice may not be removed or altered from any source distribution.

## FFmpeg/FFprobe

The pinned Gyan FFmpeg 9.0.2 essentials build is GPLv3. It is not redistributed
in this package because the corresponding-source, build-configuration, and
third-party source/notices obligations have not been cleared for redistribution.
When needed, VCX offers the pinned official download only after explicit user
consent and verifies its manifest-pinned size and SHA-256 digest before use.
See `docs/third-party/runtime-assets.md` for provenance and the release gate.
