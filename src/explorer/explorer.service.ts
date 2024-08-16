import { singleton } from "tsyringe";
import axios, { AxiosResponse } from "axios";
import { callbackUrl, explorerToken, explorerUrl } from "../config";

export interface TransactionData {
  data: {
    value: {
      wei: string,
      ether: string,
      symbol?: string
    };
    token: {
      "address": string,
      "name": string,
      "symbol": string,
      "decimals": number,
      "totalSupply": number
    };
    type: string;
    from: string;
    to: string;
    timestamp: string;
  }[];
}

@singleton()
export class ExplorerService {
  async initService() {
    await axios.post(`${explorerUrl}/watch`, { "id": explorerToken, "action": "init", "url": callbackUrl });
  }

  async subscribeAddresses(addresses: string[]) {
    await axios.post(`${explorerUrl}/watch`, { "id": explorerToken, "addresses": addresses, "action": "subscribe" });
  }

  async unsubscribeAddresses(addresses: string[]) {
    await axios.post(`${explorerUrl}/watch`, { "id": explorerToken, "addresses": addresses, "action": "unsubscribe" });
  }

  async getTransactionData(txHash: string): Promise<AxiosResponse<TransactionData>> {
    return await axios.get(`${explorerUrl}/transactions/${txHash}`);
  }

  async checkService() {
    const maxRetries = 6;

    const attempt = async (retries: number): Promise<void> => {
      try {
        await axios.post(`${explorerUrl}/watch`, { "id": explorerToken, "action": "check" });
        await this.sleep(30000);
        await attempt(maxRetries);
      } catch (error) {
        console.error(error);
        if (retries > 0) {
          await this.sleep(5000);
          await attempt(retries - 1);
        }
      }
    };

    await attempt(maxRetries);
  }

  async sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
