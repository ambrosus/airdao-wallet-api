import axios from "axios";
import {callbackUrl, explorerToken, explorerUrl} from "../config";

export class ExplorerService {
    async initService() {
        await axios.post(`${explorerUrl}/watch`, {"id": explorerToken, "action": "init", "url": callbackUrl});
    }

    async subscribeAddresses(addresses: string[]) {
        await axios.post(`${explorerUrl}/watch`, {"id": explorerToken, "addresses": addresses, "action": "subscribe"});
    }

    async unsubscribeAddresses(addresses: string[]) {
        await axios.post(`${explorerUrl}/watch`, {"id": explorerToken, "addresses": addresses, "action": "unsubscribe"});
    }

    async checkService() {

    }
}
