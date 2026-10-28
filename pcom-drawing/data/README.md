# pCOM 8비트 사양 프리셋

`presets.json`에는 기본 사양 예시 1개와 사용자가 제공한 PPT의 18개 슬라이드에 해당하는 18개 프리셋이 있습니다. 총 19개입니다. 동명 SU 파일의 V1.0/V1.1을 별도로 보존했습니다.

각 항목은 `Preset` 구조(`id`, `label`, `config`, 선택적 `source`, `notes`)를 따릅니다. 날짜와 개정은 내부 Rev 표의 마지막 유효 행에서 가져왔습니다. 원본 파일명에 있는 날짜와 다르면 내부 Rev를 적용하고 차이를 notes에 남겼습니다.

## 근거

- 기본 사양: 사용자 첨부 `도면 사양 발췌.docx`의 제품 발주 코드, 코드 예시.
- 원본 도면: `/workspace/pcom-analysis/drawing-index.json`의 18개 슬라이드 원문 전사.
- 그림/표 대조 및 절단 위치: `/workspace/pcom-analysis/pCOM-8bit-comparison.md`, `/workspace/pcom-analysis/spec/pCOM-8bit-base-spec.md`.

실행파일에는 첨부 문서 전체나 기밀 이미지 대신 프리셋의 입력 사양, 출처 이름/쪽/도면번호 및 확인 사항을 포함합니다.

## 코드 처리

- `config.bit_code`는 이 8비트 프로그램의 `A8`입니다. 원본의 `08`은 `source.drawing_number`에서 유지하고 명시적 notes를 남겼습니다. 나머지 원본 코드는 바꾸지 않았습니다.
- `config.lengths`는 mm입니다. 원본 네 길이 코드 ×10mm가 각각 원문 길이표와 일치하는지 확인했습니다.
- 코드 `0`은 최종 커넥터 없음입니다. 제거 전 조립·검사 커넥터가 BOM/그림에 나타나는 것과 선재 자체의 유무는 구분합니다.
- `g/h=1`의 기본 사양 품번 `51103-0310`과 기존 BOM `51103-0300`은 자동 치환하지 않습니다.
- 프리셋 notes는 source의 확인 사항입니다. 입력값을 변경하면 그 notes가 새로운 조합을 승인하거나 실제 원도면의 핀 배선까지 자동 확정하는 의미가 되지 않습니다.

## source별 별도 속성

다음 원도면 속성을 Config의 추가 필드와 notes에서 보존합니다. 이를 조용히 공통값으로 덮어쓰지 않아야 합니다. 추가 치수는 도면번호의 네 길이 코드에 포함되지 않습니다.

- `board_pin_order`: 기본값 `24v-gnd`. SYNUS SLAVE 두 길이형 및 LW SLAVE는 `gnd-24v`(1=GND, 2=24V). BOARD 커넥터 코드만으로 극성이 정해지지 않습니다.
- MF/SYNUS/SEMES의 26핀 SLAVE: 본체 ⑤의 GND/RX2/TX2에 NC 표기가 있으며 동시에 USER 상대 핀 숫자도 있음. LW SLAVE에는 NC 표기 없음.
- SEMES MASTER: `lengths.belt/board=550/630`mm, `connector_colors.io_in/maint/belt/board=black`. B Type 핀 배치·GND 분기와 Hoist PIO1/2 라벨은 notes에도 기록합니다.
- SEMES SLAVE: `lengths.belt/board=320/480`mm, `connector_colors.maint=black`. 4핀 전원 하우징의 일부 핀만 기재된 상태는 notes에도 기록합니다. 나머지 프리셋의 추가 전원 구간 치수는 110/110mm입니다.
- SU SLAVE 검사 전 MAINT의 D-SUB BOM vs JST3 그림, SYNUS/LW의 그림/코드 vs BOM 모델 충돌.
- 900mm를 넘는 길이, 이번 사양서에 없는 GE/MF/GL/LW 제조사 코드, SLAVE의 MASTER 마킹 등 원문 불일치.

## 검증

19개 ID가 고유하고 모든 Config가 A8임을 확인했습니다. 원본 18개 각각의 Config에서 도면번호를 다시 만들었을 때, `08→A8`을 제외하고 원본 도면번호와 동일함을 확인했습니다. 날짜가 유효한 ISO 형식이고 I/O 선택값이 이번 8비트 구현의 `0/1/2/F/L` 중 하나인지 확인했습니다.
