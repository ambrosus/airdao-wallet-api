/*
 *  Copyright: Ambrosus Inc.
 *  Email: tech@ambrosus.io
 *
 *  This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0. If a copy of the MPL was not distributed with this file, You can obtain one at https://mozilla.org/MPL/2.0/.
 *
 *  This Source Code Form is “Incompatible With Secondary Licenses”, as defined by the Mozilla Public License, v. 2.0.
 */
import { Interface, JsonRpcProvider } from "ethers";
import { rpcUrl } from "../config";

const provider = new JsonRpcProvider(rpcUrl);

const abi = [
  "function name() view returns (string)",
  "function symbol() view returns (string)"
];

const iface = new Interface(abi);

const nameData = iface.encodeFunctionData("name");
const symbolData = iface.encodeFunctionData("symbol");

export async function isERC20Standard(tokenAddress: string): Promise<boolean> {
  try {
    const nameResult = await provider.call({ to: tokenAddress, data: nameData });
    const symbolResult = await provider.call({ to: tokenAddress, data: symbolData });

    const name = iface.decodeFunctionResult("name", nameResult)[0];
    const symbol = iface.decodeFunctionResult("symbol", symbolResult)[0];

    return true;
  } catch (e) {
    console.log(e);
    return false;
  }
}

