# 框架

    Runner (运行时引擎，管理会话、状态、记忆)

        Agent （智能体）
            用 Model 思考，用 Tools 执行任务

            Model （大模型）     Tools （工具）
            Gemini              API调用
            Claude              数据库查询
            Custom              执行任务，返回结果
                                Search
总结：**Agent 是大脑，Model 是思考引擎，Tools 是手脚。**            


## Workflow Agent 总览

WorkflowAgent（编排器，确定性）

  ├── SequentialAgent  → 按顺序一步步执行

  ├── ParallelAgent    → 同时执行多个子 Agent

  └── LoopAgent        → 循环执行直到满足条件