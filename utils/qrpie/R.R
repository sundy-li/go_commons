
#读csv文件，文件路径请按自身需求
data = read.csv('/Users/coyte/go/src/qrpie/train_data.csv')

#抽样
num = length(data[,1])
train = sample(1:num,(num/3)*2)

#训练集和验证机
train_data = data[train,]
test_data = data[-train,]

#转化为矩阵数据准备训练
x = model.matrix(y~.,data)[,-1]
y = data$y


#训练
xgb.fit = xgboost(data = x, label =  y, nrounds = 7 ,
                  objective = "binary:logistic",max_depth = 2, eta = 1,eval_metric ="auc")

#cross validation训练
cv.fit = xgb.cv(max_depth = 2, eta = 1 ,data = x[train,],
                label =  y[train], metrics = "auc", nfold = 10,
                nrounds = 50,objective = "binary:logistic")

#计算概率
xgb.probs = predict(xgb.fit , x[-train,])

#查看频数分布图
hist(xgb.probs)

#设定阈值为0.2
xgb.pred = xgb.probs>0.2

#计算召回率和精确率
tp = sum(y[-train]==1&xgb.pred==1)
precision = tp/sum(xgb.pred)
recall = tp/sum(y[-train])


#画出决策树图
xgb.plot.tree(colnames(data)[-903],model = xgb.fit)

#输出特征重要性
xgb.importance(colnames(data),model=xgb.fit)

#输出决策树参数
para = xgb.model.dt.tree(colnames(data),model = xgb.fit)

#将决策树参数写入到csv文件
write.csv(para,"model.csv",row.names = FALSE)