CREATE TABLE "public"."order_details" (
  "id" bigint NOT NULL,
  "trade_date" character varying NOT NULL DEFAULT ''::character varying,
  "user_id" bigint NOT NULL DEFAULT 0,
  "order_id" bigint NOT NULL DEFAULT 0,
  "currency" character varying NOT NULL DEFAULT ''::character varying,
  "trade_amount" numeric NOT NULL DEFAULT 0.0,
  "trade_quantity" numeric NOT NULL DEFAULT 0.0,
  "order_status" character varying NOT NULL DEFAULT ''::character varying,
  "fee" numeric NOT NULL DEFAULT 0.0
);
CREATE INDEX order_details_idx ON public.order_details USING btree (trade_date, user_id, order_id);
ALTER TABLE ONLY "public"."order_details"
    ADD CONSTRAINT "order_details_pkey" PRIMARY KEY ("id");


COMMENT ON COLUMN "public"."order_details"."id" IS '自增ID';



COMMENT ON COLUMN "public"."order_details"."trade_date" IS '交易日期（2006-01-02）';


COMMENT ON COLUMN "public"."order_details"."user_id" IS '用户id';


COMMENT ON COLUMN "public"."order_details"."order_id" IS '订单ID';

COMMENT ON COLUMN "public"."order_details"."order_status" IS '订单状态：init-初始化，pending-待处理，processing-处理中，completed-已完成，cancelled-已取消';


COMMENT ON COLUMN "public"."order_details"."trade_amount" IS '成交金额';

COMMENT ON COLUMN "public"."order_details"."trade_quantity" IS '成交数量';

COMMENT ON TABLE "public"."order_details" IS '订单详情表';