# pCOM Drawing 8BIT · Windows 다운로드

이 브랜치는 pCOM 8비트 케이블 도면 작성 프로그램을 배포합니다.

## 프로그램 받기

**[Windows 프로그램 ZIP 다운로드](https://github.com/yssim44-arch/chamber-booking/raw/refs/heads/pcom-drawing-v1/pcom-drawing/downloads/pCOM-8bit-Windows-x64.zip)**

1. 위 링크에서 ZIP 파일을 받습니다.
2. ZIP 압축을 풀고 `pCOM-8bit.exe`를 실행합니다.
3. 기존 도면 프리셋 또는 기본 사양을 선택하고 **PPTX 도면 내보내기**를 누릅니다.

Windows 10/11 64비트용입니다. Python, Git, Go를 설치할 필요가 없습니다. Edge가 있으면 전용 창, 그렇지 않으면 기본 브라우저에서 화면이 열립니다.

링크가 미리보기 화면으로 열리면 파일 화면의 **Download raw file**(다운로드 아이콘)을 누르세요. 저장소 접근 권한이 있는 GitHub 계정으로 로그인해야 할 수 있습니다.

## 포함된 파일

- [다운로드 파일 모음](pcom-drawing/downloads/)
- [예제 PPTX](pcom-drawing/downloads/pCOM-8bit-example.pptx)
- [사용법과 개발 안내](pcom-drawing/README.md)
- [소스 코드](pcom-drawing/)
- [배포 파일 SHA-256](pcom-drawing/downloads/SHA256SUMS.txt)

19개 프리셋, 도면번호 생성/해석, 커넥터와 전원 핀 순서 선택, 길이 입력, 편집 가능한 PPTX/SVG 출력과 JSON 사양 저장/불러오기를 지원합니다.

출력은 **개략 연결도와 주요 자재표**입니다. 원본 커넥터 기구 형상이나 전체 제작 BOM을 그대로 복제하는 버전은 아닙니다. 원도면의 미확정·불일치 항목은 화면과 PPTX에 남습니다.

Go 테스트, Chromium 화면, 19개 프리셋 PPTX 구조와 LibreOffice 렌더를 검증했습니다. Windows EXE는 크로스 컴파일 산출물이며 Windows GUI 직접 실행 검증은 아직 하지 않았습니다.
