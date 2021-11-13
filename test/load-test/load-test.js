import http from "k6/http";
import { check, sleep } from "k6";
import { Counter } from "k6/metrics";

import config from "./config.js";

const exceptionCounts = new Counter("exception");

export const options = {
  stages: [
    { duration: "1m", target: 30 },
    { duration: "5m", target: 30 },
    { duration: "1m", target: 0 },
  ],
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<7000"],
    exception: [{ threshold: "rate<0.1", abortOnFail: true }],
  },
};

export default () => {
  const ADDRESS = config.getRandomAddress();
  const API = config.getRandomApi();

  const URL = `${__ENV.HOST}${API}`;

  const payload = {
    row: 10,
    page: 0,
    address: ADDRESS,
  };

  const params = {
    headers: {
      "Content-Type": "application/json",
    },
  };

  try {
    const res = http.request("POST", URL, JSON.stringify(payload), params);

    // status
    check(res, {
      [`[${API}: ${ADDRESS}] http code == OK`]: (res) => res.status == 200,
    });

    // resp body
    check(res.json(), {
      [`[${API}: ${ADDRESS}] transfers length > 0`]: (jsonObj) =>
        jsonObj.data.transfers.length > 0,
    });
  } catch (err) {
    exceptionCounts.add(1);
    console.error(
      `[user=${__VU} at loop=${__ITER}] ADDRESS=${ADDRESS} err: ${err}`
    );
  }
};
