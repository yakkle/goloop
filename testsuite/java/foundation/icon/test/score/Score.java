package foundation.icon.test.score;

import foundation.icon.icx.*;
import foundation.icon.icx.data.Address;
import foundation.icon.icx.data.Bytes;
import foundation.icon.icx.data.TransactionResult;
import foundation.icon.icx.transport.jsonrpc.RpcItem;
import foundation.icon.icx.transport.jsonrpc.RpcObject;
import foundation.icon.test.common.Constants;
import foundation.icon.test.common.Utils;

import java.io.IOException;
import java.math.BigInteger;
import java.util.concurrent.TimeoutException;

public class Score {
    public static final BigInteger STEPS_DEFAULT = BigInteger.valueOf(2000000);
    public static final long DEFAULT_WAITING_TIME = 3000; // millisecond

    protected IconService service;
    protected Address scoreAddress;
    protected BigInteger nid;

    public Score(IconService service, Address scoreAddress, BigInteger nid) {
        this.service = service;
        this.scoreAddress = scoreAddress;
        this.nid = nid;
    }

    public static TransactionResult deployAndWaitResult(IconService service
            , Wallet wallet, String filePath, RpcObject params)
            throws IOException, TimeoutException {
        Bytes txHash = Utils.deployScore(service, wallet, filePath, params);
        return Utils.getTransactionResult(service, txHash, DEFAULT_WAITING_TIME);
    }

    public static Address mustDeploy(IconService service, Wallet wallet, String filePath, RpcObject params)
            throws IOException, TransactionFailureException, TimeoutException {
        TransactionResult result = deployAndWaitResult(service, wallet, filePath, params);
        if (!Constants.STATUS_SUCCESS.equals(result.getStatus())) {
            throw new TransactionFailureException(result.getFailure());
        }
        return new Address(result.getScoreAddress());
    }

    public RpcItem call(Address from, String method, RpcObject params)
            throws IOException {
        if (params == null) {
            params = new RpcObject.Builder().build();
        }
        Call<RpcItem> call = new Call.Builder()
                .from(from)
                .to(this.scoreAddress)
                .method(method)
                .params(params)
                .build();
        return this.service.call(call).execute();
    }

    public Bytes invoke(Wallet wallet, String method,
                        RpcObject params, BigInteger value, BigInteger steps)
            throws IOException {
        TransactionBuilder.Builder builder = TransactionBuilder.newBuilder()
                .nid(nid)
                .from(wallet.getAddress())
                .to(this.scoreAddress)
                .stepLimit(steps);

        if ((value != null) && value.bitLength() != 0) {
            builder = builder.value(value);
        }

        Transaction t = null;
        if (params != null) {
            t = builder.call(method).params(params).build();
        } else {
            t = builder.call(method).build();
        }

        return this.service
                .sendTransaction(new SignedTransaction(t, wallet))
                .execute();
    }

    public TransactionResult invokeAndWaitResult(Wallet wallet, String method,
                                                 RpcObject params, BigInteger value, BigInteger steps)
            throws IOException, TimeoutException {
        Bytes txHash = this.invoke(wallet, method, params, value, steps);
        return waitResult(txHash);
    }

//    public Bytes transfer(Wallet wallet, Address to, BigInteger value, BigInteger steps)
//            throws IOException {
//        Transaction tx = TransactionBuilder.newBuilder()
//                .nid(Env.nodes[0].chains[0].networkId)
//                .from(wallet.getAddress())
//                .to(this.scoreAddress)
//                .value(value)
//                .stepLimit(steps)
//                .build();
//        SignedTransaction signed =
//                new SignedTransaction(tx, wallet);
//        return this.service
//                .sendTransaction(signed).execute();
//    }
//
//    public Bytes transfer(Wallet wallet, Address to, BigInteger value) throws IOException {
//        return transfer(wallet, to, value, STEPS_TRANSFER);
//    }
//
//    public TransactionResult transferAndWaitResult(Wallet wallet, Address to,
//                                                   BigInteger value, BigInteger steps
//    ) throws IOException, TransactionFailureException {
//        Bytes txHash = this.transfer(wallet, to, value, steps);
//        return waitResult(txHash);
//    }
//
//    public TransactionResult transferAndWaitResult(Wallet wallet, Address to, BigInteger value)
//        throws IOException, TransactionFailureException {
//        return transferAndWaitResult(wallet, to,  value, STEPS_TRANSFER);
//    }

    public TransactionResult waitResult(Bytes txHash) throws IOException, TimeoutException {
        return Utils.getTransactionResult(this.service, txHash, DEFAULT_WAITING_TIME);
    }

    public TransactionResult waitResult(Bytes txHash, long waiting) throws IOException, TimeoutException {
        return Utils.getTransactionResult(this.service, txHash, waiting);
    }

    public Address getAddress() {
        return this.scoreAddress;
    }

    @Override
    public String toString() {
        return "SCORE(" + this.scoreAddress.toString() + ")";
    }
}
