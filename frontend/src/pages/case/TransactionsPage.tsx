import { money } from "../../lib/claims";
import { stamp, useCaseFrame } from "./shared";

export function TransactionsPage() {
  const { detail } = useCaseFrame();
  return (
    <section className="case-section">
      <h2>Transactions</h2>
      {detail.transactions.length === 0 ? (
        <p>No transactions are linked to this case.</p>
      ) : (
        <div className="table-scroll">
          <table className="transaction-table">
            <thead>
              <tr>
                <th>Merchant / date</th>
                <th>Amount</th>
                <th>Status / context</th>
              </tr>
            </thead>
            <tbody>
              {detail.transactions.map((tx) => (
                <tr key={tx.id}>
                  <td>
                    <strong>{tx.merchant_name || "No merchant"}</strong>
                    <small>{stamp(tx.occurred_at)}</small>
                  </td>
                  <td className="amount">{money(tx.amount_minor, tx.currency)}</td>
                  <td>
                    <span className="tag">{tx.status}</span>
                    <small>
                      {tx.device_external_id || "No device"} · {tx.location_label || "No location"}
                    </small>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
