import { randomIntBetween } from "https://jslib.k6.io/k6-utils/1.1.0/index.js";

const addressList = [
  "138Mnv2ciUAXXGdLFgU6ebAibm6SKdMHCk1y6kwMmc4R18rQ", // 660 transfers
  "15guxdSyuh6HThsxTX8LhAF3mF7minL4VWBuEqQsgNkSP8ci", // 1331 transfers
  "163YfVTEqSGcYtiNjKsPWExYAFC7cthPnGrcREMmVYmJh4GS", // 2605 transfers
  "13NfRmMpbnhvpp2ZvQ4goxBoKoKd7452i9WNtQsn9oihyG8E", // 2037 transfers
  "13x3ujrh3wbHCdgJ9tuykAfsMzvzonuBgFNWD9T2C61oeHdF", // 2398 transfers
  "15jZjjX8euzphAMYkkab9yA6FTZoFyozwWBoHMSDQ5CdiTFY", // 2476 transfers
  "15i5cwqFws9EhB4tAWy3A1P4YFdri4znFjNcmV3AQUz2V2fp", // 2774 transfers
  "14priV85dNut4Vfk6h59LV3DUMcGk8VrRQCinFQXXSEAGNNk", // 2154 transfers
  "12BFQrjL4DRsgMBNajEiwtAeKnnneXr8wUKi31DVjSXnQm56", // 3030 transfers
  "1uZARZUtfF746wVLUiYrjPMxSfQqoGAQ9eFb1E4VAx5sXFT", // 3653 transfers
];

const addressList2 = [
  "144HGaYrSdK3543bi26vT6Rd8Bg7pLPMipJNr2WLc3NuHgD2", // 18943 transfers
  "157PD8GV7pJNMwN2zCEuchRyPCMeRoVtwzNdry4XjedkB2KR", // 19205 transfers
  "129fZmvLJcVXJa36p9jmvJWgEFpfvgJXNVDB91fB6FcbTEwb", // 19216 transfers
  "133SDz9BYXmVzbo7DXtXzhbUDsHLf2pY76U29m93Htm2mE8x", // 19830 transfers
  "148fP7zCq1JErXCy92PkNam4KZNcroG9zbbiPwMB1qehgeT4", // 18257 transfers
  "12k6zoi7L6Jd2oCVek7Zktj8CHY2yoY6nxZThDZK7mdWP6Sr", // 19641 transfers
  "1UbTddpy3RggGy3nk1vAc3msmSBasbhiYRQZnAdvNdUSXJn", // 34342 transfers
  "1BAitQj5xequxDyKaB1tqQ3D1k3vdSUM3geGfbSDnv87pX1", // 24518 transfers
  "1UbTddpy3RggGy3nk1vAc3msmSBasbhiYRQZnAdvNdUSXJn", // 34364 transfers
  "12yc5VAj5X6rwAWB688EtAzNTncx8Ce4nP79jW1rdJkaeNEJ", // 32135 transfers
];

const apiList = [
  "/api/scan/transfers",
  // "/api/scan/account/reward_slash",
  // "/api/wallet/bond_list",
  // "/api/open/account/extrinsics",
];

const getRandomAddress = () =>
  __ENV.HARD_MODE == 1
    ? addressList2[randomIntBetween(0, addressList2.length - 1)]
    : addressList[randomIntBetween(0, addressList.length - 1)];

const getRandomApi = () => apiList[randomIntBetween(0, apiList.length - 1)];

module.exports = {
  getRandomAddress,
  getRandomApi,
};
